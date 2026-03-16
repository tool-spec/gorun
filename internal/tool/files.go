package tool

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/hydrocode-de/gorun/internal/files"
)

func (t *Tool) ListResults() ([]files.ResultFile, error) {
	if t.Status != "finished" && t.Status != "errored" {
		return nil, errors.New("unfinished tools cannot list results")
	}

	hostOut, ok := t.Mounts["/out"]
	if !ok {
		return nil, fmt.Errorf("tool %v did not mount /out. That means there is no folder with results", t.Name)
	}
	return files.ReadDir(hostOut, true, hostOut)
}

func (t *Tool) resolveResultFile(resultPath string) (*files.ResultFile, error) {
	results, err := t.ListResults()
	if err != nil {
		return nil, err
	}

	normalized := filepath.ToSlash(path.Clean(strings.TrimSpace(resultPath)))
	normalized = strings.TrimPrefix(normalized, "./")
	if normalized == "." || normalized == "" {
		return nil, fmt.Errorf("the result file %s was not found in the tool %s results", resultPath, t.Name)
	}

	for _, file := range results {
		if filepath.ToSlash(file.RelPath) == normalized {
			matchedFile := file
			return &matchedFile, nil
		}
	}

	return nil, fmt.Errorf("the result file %s was not found in the tool %s results", resultPath, t.Name)
}

type WriteFileMeta struct {
	Filename string
	MimeType string
	FullPath string
}

func (t *Tool) WriteResultFile(resultPath string, w io.Writer) (*WriteFileMeta, error) {
	result, err := t.resolveResultFile(resultPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(result.AbsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	mimeType := http.DetectContentType(buffer)
	_, err = file.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(w, file)
	if err != nil {
		return nil, err
	}

	return &WriteFileMeta{
		Filename: path.Base(result.RelPath),
		MimeType: mimeType,
		FullPath: result.AbsPath,
	}, nil
}

type PreviewResultFileMeta struct {
	Filename  string `json:"filename"`
	MimeType  string `json:"mimeType"`
	Encoding  string `json:"encoding"`
	Truncated bool   `json:"truncated"`
	Content   string `json:"content"`
}

var previewableExtensions = []string{".json", ".txt", ".log", ".md", ".csv"}

const previewByteLimit = 64 * 1024

func (t *Tool) PreviewResultFile(resultPath string) (*PreviewResultFileMeta, error) {
	result, err := t.resolveResultFile(resultPath)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(result.AbsPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, previewByteLimit+1)
	readBytes, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	contentBytes := buffer[:readBytes]
	truncated := false
	if len(contentBytes) > previewByteLimit {
		contentBytes = contentBytes[:previewByteLimit]
		truncated = true
	}

	mimeType := http.DetectContentType(contentBytes)
	extension := strings.ToLower(path.Ext(result.RelPath))
	if !strings.HasPrefix(mimeType, "text/") && mimeType != "application/json" && !slices.Contains(previewableExtensions, extension) {
		return nil, fmt.Errorf("preview is not available for binary or unsupported file type: %s", result.RelPath)
	}
	if !utf8.Valid(contentBytes) {
		return nil, fmt.Errorf("preview is not available for binary or unsupported file type: %s", result.RelPath)
	}

	return &PreviewResultFileMeta{
		Filename:  result.RelPath,
		MimeType:  mimeType,
		Encoding:  "utf-8",
		Truncated: truncated,
		Content:   string(contentBytes),
	}, nil
}
