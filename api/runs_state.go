package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/tool"
	"github.com/spf13/viper"
)

type RunsResponse struct {
	Count  int           `json:"count"`
	Status string        `json:"status"`
	Runs   []RunListItem `json:"runs"`
}

type RunDetailResponse struct {
	tool.Tool
	GotapMetadata interface{} `json:"gotap_metadata,omitempty"`
}

type RunResultSummary struct {
	ArtifactCount int   `json:"artifact_count"`
	LogCount      int   `json:"log_count"`
	InternalCount int   `json:"internal_count"`
	MetadataCount int   `json:"metadata_count"`
	TotalSize     int64 `json:"total_size"`
}

type RunListItem struct {
	tool.Tool
	GotapMetadata interface{}       `json:"gotap_metadata,omitempty"`
	ResultSummary *RunResultSummary `json:"result_summary,omitempty"`
}

func classifyResultFile(name string) string {
	switch name {
	case "_metadata.json":
		return "metadata"
	case "STDOUT.log", "STDERR.log":
		return "log"
	}
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return "internal"
	}
	return "artifact"
}

func summarizeResults(run tool.Tool) *RunResultSummary {
	results, err := run.ListResults()
	if err != nil {
		return nil
	}

	summary := &RunResultSummary{}
	for _, result := range results {
		switch classifyResultFile(result.Name) {
		case "artifact":
			summary.ArtifactCount++
		case "log":
			summary.LogCount++
		case "metadata":
			summary.MetadataCount++
		case "internal":
			summary.InternalCount++
		}
		summary.TotalSize += result.Size
	}

	return summary
}

func appendMetadataFields(run db.Run, item *RunListItem) {
	if !run.GotapMetadata.Valid {
		return
	}

	var metadata interface{}
	if err := json.Unmarshal([]byte(run.GotapMetadata.String), &metadata); err != nil {
		log.Printf("failed parsing gotap metadata for run %d: %v", run.ID, err)
		return
	}
	item.GotapMetadata = metadata
}

func GetAllRuns(w http.ResponseWriter, r *http.Request) {
	filter := r.URL.Query().Get("status")
	DB := viper.Get("db").(*db.Queries)

	user_id := r.Header.Get("X-User-ID")
	log.Printf("user_id: %s", user_id)
	log.Printf("r.Header: %v", r.Header)
	if user_id == "" {
		RespondWithError(w, http.StatusUnauthorized, "User ID is required")
		return
	}

	var runs []db.Run
	var err error
	switch filter {
	case "pending":
		runs, err = DB.GetIdleRuns(r.Context(), db.GetIdleRunsParams{
			UserID: user_id,
		})
	case "running":
		runs, err = DB.GetRunning(r.Context(), db.GetRunningParams{
			UserID: user_id,
		})
	case "finished":
		runs, err = DB.GetFinishedRuns(r.Context(), db.GetFinishedRunsParams{
			UserID: user_id,
		})
	case "errored":
		runs, err = DB.GetErroredRuns(r.Context(), db.GetErroredRunsParams{
			UserID: user_id,
		})
	default:
		runs, err = DB.GetAllRuns(r.Context(), db.GetAllRunsParams{
			UserID: user_id,
		})
	}
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}

	var toolRuns []RunListItem
	for _, dbRun := range runs {
		toolRun, err := tool.FromDBRun(dbRun)
		if err != nil {
			log.Printf("Error while loading tool run: %s", err)
			continue
		}
		item := RunListItem{Tool: toolRun}
		appendMetadataFields(dbRun, &item)
		item.ResultSummary = summarizeResults(toolRun)
		toolRuns = append(toolRuns, item)
	}

	RespondWithJSON(w, http.StatusOK, RunsResponse{
		Count:  len(runs),
		Status: filter,
		Runs:   toolRuns,
	})
}

func DeleteRun(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	user_id := r.Header.Get("X-User-ID")
	if user_id == "" {
		RespondWithError(w, http.StatusUnauthorized, "User ID is required")
		return
	}
	DB := viper.Get("db").(*db.Queries)

	// the tool may have a saved mount point, so we delete it first
	_, ok := tool.Mounts["/in"]
	if ok {
		parent := filepath.Dir(tool.Mounts["/in"])
		err := os.RemoveAll(parent)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
		}

	}

	err := DB.DeleteRun(r.Context(), db.DeleteRunParams{
		ID:     tool.ID,
		UserID: user_id,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}
	RespondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Run deleted",
	})

}

func GetRunStatus(w http.ResponseWriter, r *http.Request, run tool.Tool) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		RespondWithError(w, http.StatusUnauthorized, "User ID is required")
		return
	}
	DB := viper.Get("db").(*db.Queries)

	dbRun, err := DB.GetRun(r.Context(), db.GetRunParams{
		ID:     run.ID,
		UserID: userID,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := RunDetailResponse{Tool: run}
	if dbRun.GotapMetadata.Valid {
		var metadata interface{}
		if err := json.Unmarshal([]byte(dbRun.GotapMetadata.String), &metadata); err != nil {
			log.Printf("failed parsing gotap metadata for run %d: %v", run.ID, err)
		} else {
			resp.GotapMetadata = metadata
		}
	}

	RespondWithJSON(w, http.StatusOK, resp)
}

func HandleRunStart(w http.ResponseWriter, r *http.Request, run tool.Tool) {
	user_id := r.Header.Get("X-User-ID")
	if user_id == "" {
		RespondWithError(w, http.StatusUnauthorized, "User ID is required")
		return
	}
	DB := viper.Get("db").(*db.Queries)

	opt := tool.RunToolOptions{
		DB:   DB,
		Tool: run,
		Env:  []string{},
		// Cmd:  []string{},
		UserId: user_id,
	}

	go tool.RunTool(context.Background(), opt)

	// wait a few miliseconds to make sure the container is started
	time.Sleep(time.Millisecond * 100)
	started, err := DB.GetRun(r.Context(), db.GetRunParams{
		ID:     run.ID,
		UserID: user_id,
	})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}
	RespondWithJSON(w, http.StatusProcessing, started)
}
