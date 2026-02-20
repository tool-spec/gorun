package mcp

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/hydrocode-de/gorun/internal/auth"
	"github.com/hydrocode-de/gorun/internal/service"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/viper"
)

type contextKey string

const userIDContextKey contextKey = "mcp_user_id"

type Config struct {
	AuthRequired   bool
	InsecureNoAuth bool
}

type Server struct {
	svc               *service.Service
	cfg               Config
	mcpServer         *server.MCPServer
	stdioServer       *server.StdioServer
	httpServer        *server.StreamableHTTPServer
	warnedInsecureHit atomic.Bool
}

func NewServer(svc *service.Service, cfg Config) *Server {
	s := &Server{svc: svc, cfg: cfg}
	s.mcpServer = server.NewMCPServer("gorun-mcp", "0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)
	s.registerTools()
	s.registerResources()
	s.stdioServer = server.NewStdioServer(s.mcpServer)
	s.stdioServer.SetContextFunc(func(ctx context.Context) context.Context {
		credentials, err := auth.GetAdminCredentials(ctx)
		if err != nil {
			return context.WithValue(ctx, userIDContextKey, "")
		}
		return context.WithValue(ctx, userIDContextKey, credentials.UserID)
	})
	s.httpServer = server.NewStreamableHTTPServer(s.mcpServer,
		server.WithHTTPContextFunc(func(ctx context.Context, r *http.Request) context.Context {
			userID := ""
			if id, ok := s.authenticateHTTP(r); ok {
				userID = id
			}
			return context.WithValue(ctx, userIDContextKey, userID)
		}),
	)
	return s
}

func (s *Server) HTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.authenticateHTTP(r); !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		s.httpServer.ServeHTTP(w, r)
	})
}

func (s *Server) RunStdio(ctx context.Context, stdin io.Reader, stdout io.Writer) error {
	return s.stdioServer.Listen(ctx, stdin, stdout)
}

func (s *Server) authenticateHTTP(r *http.Request) (string, bool) {
	if !s.cfg.AuthRequired || s.cfg.InsecureNoAuth {
		if s.cfg.InsecureNoAuth && !s.warnedInsecureHit.Swap(true) {
			log.Printf("WARNING: MCP HTTP insecure-no-auth mode is enabled")
		}
		credentials, err := auth.GetAdminCredentials(r.Context())
		if err != nil {
			return "", false
		}
		return credentials.UserID, true
	}
	authHeader := r.Header.Get("Authorization")
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" || token == authHeader {
		return "", false
	}
	userID, err := auth.ValidateJWT(token, viper.GetString("secret"))
	if err != nil {
		return "", false
	}
	return userID, true
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(mcpgo.NewTool("run_tool",
		mcpgo.WithDescription("Validate, create, and start a gorun tool in one call"),
		mcpgo.WithString("tool_name", mcpgo.Required(), mcpgo.Description("Tool name from tool spec")),
		mcpgo.WithString("docker_image", mcpgo.Required(), mcpgo.Description("Docker image containing the tool")),
		mcpgo.WithObject("parameters", mcpgo.Description("Tool parameters object")),
		mcpgo.WithObject("data", mcpgo.Description("Input dataset name to host path mapping")),
		mcpgo.WithString("client_request_id", mcpgo.Description("Optional client request id")),
	), s.handleRunTool)

	s.mcpServer.AddTool(mcpgo.NewTool("get_run",
		mcpgo.WithDescription("Get run status and metadata"),
		mcpgo.WithNumber("run_id", mcpgo.Required()),
	), s.handleGetRun)

	s.mcpServer.AddTool(mcpgo.NewTool("list_run_results",
		mcpgo.WithDescription("List result files for a run"),
		mcpgo.WithNumber("run_id", mcpgo.Required()),
	), s.handleListRunResults)

	s.mcpServer.AddTool(mcpgo.NewTool("get_run_result_file",
		mcpgo.WithDescription("Read a result file for a run"),
		mcpgo.WithNumber("run_id", mcpgo.Required()),
		mcpgo.WithString("filename", mcpgo.Required()),
	), s.handleGetRunResultFile)

	s.mcpServer.AddTool(mcpgo.NewTool("list_specs",
		mcpgo.WithDescription("List cached tool specs"),
		mcpgo.WithString("filter"),
	), s.handleListSpecs)

	s.mcpServer.AddTool(mcpgo.NewTool("get_spec",
		mcpgo.WithDescription("Get a tool spec by slug"),
		mcpgo.WithString("tool_slug", mcpgo.Required()),
	), s.handleGetSpec)
}

func (s *Server) registerResources() {
	s.mcpServer.AddResource(
		mcpgo.NewResource("spec://{toolSlug}", "Tool spec", mcpgo.WithResourceDescription("Resolve a cached tool specification"), mcpgo.WithMIMEType("application/json")),
		s.handleResourceRead,
	)
	s.mcpServer.AddResource(
		mcpgo.NewResource("run://{id}/status", "Run status", mcpgo.WithResourceDescription("Current run status and metadata"), mcpgo.WithMIMEType("application/json")),
		s.handleResourceRead,
	)
	s.mcpServer.AddResource(
		mcpgo.NewResource("run://{id}/results-index", "Run results", mcpgo.WithResourceDescription("Result file listing for a run"), mcpgo.WithMIMEType("application/json")),
		s.handleResourceRead,
	)
}

func (s *Server) handleRunTool(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	userID := userIDFromContext(ctx)
	payload := service.CreateRunInput{
		ToolName:    req.GetString("tool_name", ""),
		DockerImage: req.GetString("docker_image", ""),
		Parameters:  toInterfaceMap(req.GetArguments()["parameters"]),
		DataPaths:   toStringMap(req.GetArguments()["data"]),
	}
	payload.ClientRequestID = req.GetString("client_request_id", "")

	result, err := s.svc.CreateAndStartRun(ctx, userID, payload)
	if err != nil {
		return s.toolError(err), nil
	}
	status := result.Run.Status
	if result.StartFailed {
		status = "error_start_failed"
	}
	out := map[string]interface{}{
		"run_id":        result.Run.ID,
		"status":        status,
		"created_at":    result.Run.CreatedAt,
		"started_at":    result.Run.StartedAt,
		"resource_uris": []string{fmt.Sprintf("run://%d/status", result.Run.ID), fmt.Sprintf("run://%d/results-index", result.Run.ID)},
		"next_steps":    []string{"call get_run with run_id", "call list_run_results once status is finished"},
	}
	if result.StartFailed {
		out["start_error"] = result.StartError
	}
	return toolJSONResult(out)
}

func (s *Server) handleGetRun(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	runID, err := parseRunIDArg(req.GetArguments()["run_id"])
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	detail, err := s.svc.GetRunDetail(ctx, userIDFromContext(ctx), runID)
	if err != nil {
		return s.toolError(err), nil
	}
	return toolJSONResult(detail)
}

func (s *Server) handleListRunResults(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	runID, err := parseRunIDArg(req.GetArguments()["run_id"])
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	results, err := s.svc.ListRunResults(ctx, userIDFromContext(ctx), runID)
	if err != nil {
		return s.toolError(err), nil
	}
	return toolJSONResult(map[string]interface{}{"count": len(results), "files": results})
}

func (s *Server) handleGetRunResultFile(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	runID, err := parseRunIDArg(req.GetArguments()["run_id"])
	if err != nil {
		return mcpgo.NewToolResultError(err.Error()), nil
	}
	result, err := s.svc.GetResultFile(ctx, userIDFromContext(ctx), runID, req.GetString("filename", ""))
	if err != nil {
		return s.toolError(err), nil
	}
	return toolJSONResult(map[string]interface{}{
		"filename":       result.Meta.Filename,
		"mime_type":      result.Meta.MimeType,
		"path":           result.Meta.FullPath,
		"content_base64": base64.StdEncoding.EncodeToString(result.Content),
	})
}

func (s *Server) handleListSpecs(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	specs := s.svc.ListToolSpecs(req.GetString("filter", ""))
	return toolJSONResult(map[string]interface{}{"count": len(specs), "tools": specs})
}

func (s *Server) handleGetSpec(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
	spec, err := s.svc.GetToolSpec(req.GetString("tool_slug", ""))
	if err != nil {
		return s.toolError(err), nil
	}
	return toolJSONResult(spec)
}

func (s *Server) handleResourceRead(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
	uri := req.Params.URI
	userID := userIDFromContext(ctx)
	var out interface{}
	switch {
	case strings.HasPrefix(uri, "spec://"):
		slug := strings.TrimPrefix(uri, "spec://")
		spec, err := s.svc.GetToolSpec(slug)
		if err != nil {
			return nil, err
		}
		out = spec
	case strings.HasPrefix(uri, "run://") && strings.HasSuffix(uri, "/status"):
		runID, err := parseRunIDFromURI(uri, "/status")
		if err != nil {
			return nil, err
		}
		detail, err := s.svc.GetRunDetail(ctx, userID, runID)
		if err != nil {
			return nil, err
		}
		out = detail
	case strings.HasPrefix(uri, "run://") && strings.HasSuffix(uri, "/results-index"):
		runID, err := parseRunIDFromURI(uri, "/results-index")
		if err != nil {
			return nil, err
		}
		results, err := s.svc.ListRunResults(ctx, userID, runID)
		if err != nil {
			return nil, err
		}
		out = map[string]interface{}{"count": len(results), "files": results}
	default:
		return nil, fmt.Errorf("resource not found: %s", uri)
	}
	content, _ := json.Marshal(out)
	return []mcpgo.ResourceContents{mcpgo.TextResourceContents{URI: uri, MIMEType: "application/json", Text: string(content)}}, nil
}

func (s *Server) toolError(err error) *mcpgo.CallToolResult {
	if err == nil {
		return mcpgo.NewToolResultError("unknown error")
	}
	if ve, ok := service.IsValidationError(err); ok {
		payload, _ := json.Marshal(map[string]interface{}{"message": ve.Message, "errors": ve.Errors})
		return mcpgo.NewToolResultError(string(payload))
	}
	return mcpgo.NewToolResultError(err.Error())
}

func userIDFromContext(ctx context.Context) string {
	v := ctx.Value(userIDContextKey)
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func parseRunIDFromURI(uri string, suffix string) (int64, error) {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(uri, "run://"), suffix)
	trimmed = strings.TrimSuffix(trimmed, "/")
	return strconv.ParseInt(trimmed, 10, 64)
}

func parseRunIDArg(v interface{}) (int64, error) {
	switch t := v.(type) {
	case float64:
		return int64(t), nil
	case int64:
		return t, nil
	case int:
		return int64(t), nil
	case string:
		return strconv.ParseInt(t, 10, 64)
	default:
		return 0, fmt.Errorf("invalid run_id")
	}
}

func toInterfaceMap(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return map[string]interface{}{}
	}
	return m
}

func toStringMap(v interface{}) map[string]string {
	out := map[string]string{}
	m, ok := v.(map[string]interface{})
	if !ok {
		return out
	}
	for k, raw := range m {
		if s, ok := raw.(string); ok {
			out[k] = s
		}
	}
	return out
}

func toolJSONResult(data interface{}) (*mcpgo.CallToolResult, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return mcpgo.NewToolResultText(string(payload)), nil
}
