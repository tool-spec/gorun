package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/hydrocode-de/gorun/internal/auth"
	"github.com/hydrocode-de/gorun/internal/frontend"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var logger = logrus.New()

func HandleApiKey(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		noAuth := viper.GetBool("no_auth")

		if noAuth {
			logger.Printf("no_auth is enabled, getting admin credentials")
			credentials, err := auth.GetAdminCredentials(r.Context())
			if err != nil {
				logger.Printf("failed to get admin credentials: %v", err)
				RespondWithError(w, http.StatusInternalServerError, "Failed to get admin credentials")
				return
			}
			logger.Printf("setting admin user ID: %s", credentials.UserID)
			r.Header.Set("X-User-ID", credentials.UserID)
			handler(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		apiKey := strings.TrimPrefix(authHeader, "Bearer ")
		secret := viper.GetString("secret")

		if apiKey != "" {
			userId, err := auth.ValidateJWT(apiKey, secret)
			if err == nil {
				r.Header.Set("X-User-ID", userId)
			}
		}

		handler(w, r)
	}
}

func CreateServer() (*http.ServeMux, error) {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// add a FileServer to serve the manager
	//mux.Handle("/manager/", http.StripPrefix("/manager/", http.FileServer(http.Dir("manager/build"))))
	mux.Handle("/manager/", http.StripPrefix("/manager/", http.FileServerFS(frontend.GetManager())))

	mux.HandleFunc("GET /runs", HandleApiKey(GetAllRuns))
	mux.HandleFunc("POST /runs", HandleApiKey(CreateRun))
	mux.HandleFunc("GET /runs/{id}", HandleApiKey(RunMiddleware(GetRunStatus)))
	mux.HandleFunc("DELETE /runs/{id}", HandleApiKey(RunMiddleware(DeleteRun)))
	mux.HandleFunc("POST /runs/{id}/start", HandleApiKey(RunMiddleware(HandleRunStart)))
	mux.HandleFunc("GET /runs/{id}/results", HandleApiKey(RunMiddleware(ListRunResults)))
	mux.HandleFunc("GET /runs/{id}/results/{filename}/preview", HandleApiKey(RunMiddleware(PreviewResultFile)))
	mux.HandleFunc("GET /runs/{id}/results/{filename}", HandleApiKey(RunMiddleware(GetResultFile)))
	mux.HandleFunc("POST /files", HandleApiKey(HandleFileUpload))
	mux.HandleFunc("GET /files", HandleApiKey(FindFile))
	mux.HandleFunc("GET /specs", ListToolSpecs)
	mux.HandleFunc("GET /specs/{toolname}", GetToolSpec)
	mux.HandleFunc("POST /auth/refresh", HandleRefreshToken)
	mux.HandleFunc("POST /auth/login", HandleLogin)
	return mux, nil
}

func RespondWithError(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(status)
	w.Write([]byte(err))
}

func RespondWithJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		w.Write([]byte(`{"error": "Failed to encode response"}`))
	}
}
