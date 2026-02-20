package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hydrocode-de/gorun/internal/cache"
	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/files"
	"github.com/hydrocode-de/gorun/internal/tool"
	toolspec "github.com/hydrocode-de/tool-spec-go"
	"github.com/hydrocode-de/tool-spec-go/validate"
)

var (
	ErrUnauthorized = errors.New("user id is required")
	ErrNotFound     = errors.New("not found")
)

type Service struct {
	DB    *db.Queries
	Cache *cache.Cache
}

type CreateRunInput struct {
	ToolName        string                 `json:"tool_name"`
	DockerImage     string                 `json:"docker_image"`
	Parameters      map[string]interface{} `json:"parameters"`
	DataPaths       map[string]string      `json:"data"`
	ClientRequestID string                 `json:"client_request_id,omitempty"`
}

type CreateAndStartResult struct {
	Run         tool.Tool `json:"run"`
	StartFailed bool      `json:"start_failed"`
	StartError  string    `json:"start_error,omitempty"`
}

type RunDetail struct {
	Tool          tool.Tool   `json:"tool"`
	GotapMetadata interface{} `json:"gotap_metadata,omitempty"`
}

type ResultFileContent struct {
	Meta    tool.WriteFileMeta `json:"meta"`
	Content []byte             `json:"content"`
}

type ValidationError struct {
	Message string      `json:"message"`
	Errors  interface{} `json:"errors"`
}

func (s *Service) ListToolSpecs(filter string) []toolspec.ToolSpec {
	specs := s.Cache.ListToolSpecs()
	if filter == "" {
		return specs
	}
	needle := strings.ToLower(filter)
	out := make([]toolspec.ToolSpec, 0, len(specs))
	for _, spec := range specs {
		if strings.Contains(strings.ToLower(spec.Name), needle) || strings.Contains(strings.ToLower(spec.Title), needle) {
			out = append(out, spec)
		}
	}
	return out
}

func (s *Service) GetToolSpec(toolSlug string) (*toolspec.ToolSpec, error) {
	spec, ok := s.Cache.GetToolSpec(toolSlug)
	if !ok {
		return nil, fmt.Errorf("%w: tool %s", ErrNotFound, toolSlug)
	}
	return spec, nil
}

func (s *Service) ValidateAndCreateRun(ctx context.Context, userID string, payload CreateRunInput) (tool.Tool, error) {
	if userID == "" {
		return tool.Tool{}, ErrUnauthorized
	}
	toolSlug := fmt.Sprintf("%s::%s", payload.DockerImage, payload.ToolName)
	toolSpec, found := s.Cache.GetToolSpec(toolSlug)
	if !found {
		return tool.Tool{}, fmt.Errorf("%w: tool %s", ErrNotFound, toolSlug)
	}
	hasErrors, errs := validate.ValidateInputs(*toolSpec, toolspec.ToolInput{Parameters: payload.Parameters, Datasets: payload.DataPaths})
	if hasErrors {
		return tool.Tool{}, ValidationError{Message: fmt.Sprintf("the provided payload is invalid for the tool %s", toolSlug), Errors: errs}
	}

	runData, err := tool.CreateToolRun(ctx, "_random", tool.CreateRunOptions{
		Name:       payload.ToolName,
		Image:      payload.DockerImage,
		Parameters: payload.Parameters,
		Datasets:   payload.DataPaths,
	}, userID)
	if err != nil {
		return tool.Tool{}, err
	}

	runTool, err := tool.FromDBRun(runData)
	if err != nil {
		return tool.Tool{}, err
	}
	return runTool, nil
}

func (s *Service) StartRun(ctx context.Context, userID string, run tool.Tool) (tool.Tool, error) {
	if userID == "" {
		return tool.Tool{}, ErrUnauthorized
	}
	opt := tool.RunToolOptions{DB: s.DB, Tool: run, Env: []string{}, UserId: userID}
	go tool.RunTool(context.Background(), opt)
	time.Sleep(100 * time.Millisecond)
	started, err := s.DB.GetRun(ctx, db.GetRunParams{ID: run.ID, UserID: userID})
	if err != nil {
		return tool.Tool{}, err
	}
	return tool.FromDBRun(started)
}

func (s *Service) CreateAndStartRun(ctx context.Context, userID string, payload CreateRunInput) (CreateAndStartResult, error) {
	runTool, err := s.ValidateAndCreateRun(ctx, userID, payload)
	if err != nil {
		return CreateAndStartResult{}, err
	}
	started, err := s.StartRun(ctx, userID, runTool)
	if err != nil {
		return CreateAndStartResult{Run: runTool, StartFailed: true, StartError: err.Error()}, nil
	}
	return CreateAndStartResult{Run: started}, nil
}

func (s *Service) GetRunDetail(ctx context.Context, userID string, runID int64) (RunDetail, error) {
	if userID == "" {
		return RunDetail{}, ErrUnauthorized
	}
	dbRun, err := s.DB.GetRun(ctx, db.GetRunParams{ID: runID, UserID: userID})
	if err != nil {
		return RunDetail{}, err
	}
	runTool, err := tool.FromDBRun(dbRun)
	if err != nil {
		return RunDetail{}, err
	}
	resp := RunDetail{Tool: runTool}
	if dbRun.GotapMetadata.Valid {
		var metadata interface{}
		if err := json.Unmarshal([]byte(dbRun.GotapMetadata.String), &metadata); err == nil {
			resp.GotapMetadata = metadata
		}
	}
	return resp, nil
}

func (s *Service) ListRunResults(ctx context.Context, userID string, runID int64) ([]files.ResultFile, error) {
	detail, err := s.GetRunDetail(ctx, userID, runID)
	if err != nil {
		return nil, err
	}
	return detail.Tool.ListResults()
}

func (s *Service) GetResultFile(ctx context.Context, userID string, runID int64, filename string) (ResultFileContent, error) {
	detail, err := s.GetRunDetail(ctx, userID, runID)
	if err != nil {
		return ResultFileContent{}, err
	}
	buf := bytes.NewBuffer(nil)
	meta, err := detail.Tool.WriteResultFile(filename, buf)
	if err != nil {
		return ResultFileContent{}, err
	}
	if meta == nil {
		return ResultFileContent{}, errors.New("result file metadata missing")
	}
	return ResultFileContent{Meta: *meta, Content: buf.Bytes()}, nil
}

func (e ValidationError) Error() string {
	return e.Message
}

func IsValidationError(err error) (ValidationError, bool) {
	var ve ValidationError
	ok := errors.As(err, &ve)
	return ve, ok
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || strings.Contains(strings.ToLower(err.Error()), "not found")
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}
