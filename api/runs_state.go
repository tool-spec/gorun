package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/tool"
	"github.com/spf13/viper"
)

type RunsResponse struct {
	Count  int         `json:"count"`
	Status string      `json:"status"`
	Runs   []tool.Tool `json:"runs"`
}

type RunDetailResponse struct {
	tool.Tool
	GotapMetadata interface{} `json:"gotap_metadata,omitempty"`
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

	var toolRuns []tool.Tool
	for _, run := range runs {
		toolRun, err := tool.FromDBRun(run)
		if err != nil {
			log.Printf("Error while loading tool run: %s", err)
			continue
		}
		toolRuns = append(toolRuns, toolRun)
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
