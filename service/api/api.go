package api

import (
	"errors"
	"log/slog"
	"net/http"
	"wasatext/service/database"
)

// Config contains the dependencies needed by the API router.
type Config struct {
	Database database.AppDatabase
	Logger   *slog.Logger
}

// Router exposes the WASAText HTTP API.
type Router interface {
	Handler() http.Handler
}

type router struct {
	db     database.AppDatabase
	mux    *http.ServeMux
	logger *slog.Logger
}

// New creates a new API router.
func New(cfg Config) (Router, error) {
	if cfg.Database == nil {
		return nil, errors.New("database is required")
	}

	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	rt := &router{
		db:     cfg.Database,
		mux:    http.NewServeMux(),
		logger: logger,
	}
	rt.registerRoutes()

	return rt, nil
}

// Handler returns the HTTP handler for all registered API routes.
func (rt *router) Handler() http.Handler {
	return rt.mux
}
