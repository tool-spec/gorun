package api

import (
	"encoding/json"
	"net/http"

	"github.com/hydrocode-de/gorun/internal/db"
	"github.com/hydrocode-de/gorun/internal/service"
	"github.com/hydrocode-de/gorun/internal/tool"
	toolspec "github.com/hydrocode-de/tool-spec-go"
)

type ListToolSpecResponse struct {
	Count int                 `json:"count"`
	Tools []toolspec.ToolSpec `json:"tools"`
}

type CreateRunPayload struct {
	ToolName    string                 `json:"name"`
	DockerImage string                 `json:"docker_image"`
	Parameters  map[string]interface{} `json:"parameters"`
	DataPaths   map[string]string      `json:"data"`
}

func RunMiddleware(handler func(http.ResponseWriter, *http.Request, tool.Tool)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		user_id := r.Header.Get("X-User-ID")
		if user_id == "" {
			RespondWithError(w, http.StatusUnauthorized, "User ID is required")
			return
		}
		DB := getService().DB

		run, err := runFromRequest(r.Context(), r, DB, user_id)
		if err != nil {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}

		tool, err := tool.FromDBRun(run)
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, err.Error())
		}

		handler(w, r, tool)
	}
}

func GetToolSpec(w http.ResponseWriter, r *http.Request) {
	toolName := r.PathValue("toolname")
	if toolName == "" {
		RespondWithError(w, http.StatusNotFound, "missing tool name")
		return
	}

	svc := getService()
	spec, err := svc.GetToolSpec(toolName)
	if err != nil {
		RespondWithError(w, http.StatusNotFound, "tool not found")
		return
	}
	RespondWithJSON(w, http.StatusOK, spec)
}

func ListToolSpecs(w http.ResponseWriter, r *http.Request) {
	svc := getService()
	specs := svc.ListToolSpecs(r.URL.Query().Get("filter"))

	RespondWithJSON(w, http.StatusOK, ListToolSpecResponse{
		Count: len(specs),
		Tools: specs,
	})
}

func CreateRun(w http.ResponseWriter, r *http.Request) {
	user_id := r.Header.Get("X-User-ID")
	if user_id == "" {
		RespondWithError(w, http.StatusUnauthorized, "User ID is required")
		return
	}

	var payload CreateRunPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	svc := getService()
	runTool, err := svc.ValidateAndCreateRun(r.Context(), user_id, service.CreateRunInput{
		ToolName:    payload.ToolName,
		DockerImage: payload.DockerImage,
		Parameters:  payload.Parameters,
		DataPaths:   payload.DataPaths,
	})
	if err != nil {
		if ve, ok := service.IsValidationError(err); ok {
			RespondWithJSON(w, http.StatusBadRequest, map[string]interface{}{
				"message": ve.Message,
				"errors":  ve.Errors,
			})
			return
		}
		if service.IsNotFound(err) {
			RespondWithError(w, http.StatusNotFound, err.Error())
			return
		}
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	DB := getService().DB
	runData, err := DB.GetRun(r.Context(), db.GetRunParams{ID: runTool.ID, UserID: user_id})
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusCreated, runData)
}
