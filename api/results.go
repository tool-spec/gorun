package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hydrocode-de/gorun/internal/files"
	"github.com/hydrocode-de/gorun/internal/tool"
)

type ListRunResultsResponse struct {
	Count int                `json:"count"`
	Files []files.ResultFile `json:"files"`
}

type PreviewResultResponse struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mimeType"`
	Encoding  string `json:"encoding"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

func resultPathFromRequest(r *http.Request) (string, error) {
	filename := r.PathValue("filename")
	if filename == "" {
		return "", fmt.Errorf("missing filename")
	}

	decoded, err := url.PathUnescape(filename)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(decoded, "/"), nil
}

func ListRunResults(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	results, err := tool.ListResults()
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
	}

	RespondWithJSON(w, http.StatusOK, ListRunResultsResponse{
		Count: len(results),
		Files: results,
	})
}

func GetResultFile(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	filename, err := resultPathFromRequest(r)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	var payload bytes.Buffer
	info, err := tool.WriteResultFile(filename, &payload)
	if err != nil {
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", info.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", info.Filename))
	_, _ = w.Write(payload.Bytes())
}

func PreviewResultFile(w http.ResponseWriter, r *http.Request, tool tool.Tool) {
	filename, err := resultPathFromRequest(r)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	preview, err := tool.PreviewResultFile(filename)
	if err != nil {
		if strings.Contains(err.Error(), "preview is not available") {
			RespondWithError(w, http.StatusUnsupportedMediaType, err.Error())
			return
		}
		RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondWithJSON(w, http.StatusOK, PreviewResultResponse{
		Filename:  preview.Filename,
		MimeType:  preview.MimeType,
		Encoding:  preview.Encoding,
		Truncated: preview.Truncated,
		Content:   preview.Content,
	})
}
