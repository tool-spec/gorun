package api

import (
	"bytes"
	"encoding/base64"
	"net/http"

	"github.com/hydrocode-de/gorun/internal/files"
	"github.com/hydrocode-de/gorun/internal/service"
	"github.com/hydrocode-de/gorun/internal/tool"
)

type ListRunResultsResponse struct {
	Count int                `json:"count"`
	Files []files.ResultFile `json:"files"`
}

func ListRunResults(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	userID := r.Header.Get("X-User-ID")
	svc := getService()
	results, err := svc.ListRunResults(r.Context(), userID, tool.ID)
	if err != nil {
		if service.IsUnauthorized(err) {
			RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusOK, ListRunResultsResponse{
		Count: len(results),
		Files: results,
	})
}

func GetResultFile(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	filename := r.PathValue("filename")
	userID := r.Header.Get("X-User-ID")
	svc := getService()
	result, err := svc.GetResultFile(r.Context(), userID, tool.ID, filename)
	if err != nil {
		if service.IsUnauthorized(err) {
			RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", result.Meta.MimeType)
	w.Header().Set("Content-Disposition", "attachment; filename="+result.Meta.Filename)
	w.Header().Set("X-Result-Path", result.Meta.FullPath)
	_, _ = bytes.NewBuffer(result.Content).WriteTo(w)
}

func encodeResultAsJSONResponse(w http.ResponseWriter, result service.ResultFileContent) {
	RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"filename":       result.Meta.Filename,
		"mime_type":      result.Meta.MimeType,
		"path":           result.Meta.FullPath,
		"content_base64": base64.StdEncoding.EncodeToString(result.Content),
	})
}
