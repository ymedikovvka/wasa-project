package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

// internalError logs the underlying cause with structured context so operators
// can diagnose 500s, then writes the generic client-facing payload. All API
// handlers should route their 500 path through here so the server log is
// never silent about an internal failure.
func (rt *router) internalError(w http.ResponseWriter, r *http.Request, where string, err error) {
	rt.logger.Error("internal server error",
		slog.String("where", where),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("error", err),
	)
	writeError(w, http.StatusInternalServerError, "internal server error")
}
