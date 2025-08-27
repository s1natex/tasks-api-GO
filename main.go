package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/s1natex/tasks-api-GO/internal/middleware"
	"github.com/s1natex/tasks-api-GO/internal/tasks"
)

// main is the entry point of the application
func main() {
	logger := newLoggerFromEnv()
	slog.SetDefault(logger)

	repo := tasks.NewInMemoryRepo()
	r := newRouter(repo, logger)

	logger.Info("server_listen", slog.String("addr", ":8080"))
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Error("server_error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// newRouter sets up the HTTP router with middleware and routes
func newRouter(repo tasks.Repository, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(15 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RequestLogger(logger))
	r.Use(middleware.MetricsMiddleware)
	// Prometheus metrics endpoint
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		middleware.MetricsHandler().ServeHTTP(w, r)
	})
	// Simple health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	tasks.RegisterRoutes(r, repo)

	return r
}

// newLoggerFromEnv creates a slog.Logger with level from LOG_LEVEL env var (debug, info, warn, error)
func newLoggerFromEnv() *slog.Logger {
	level := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn", "warning":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: l,
	})
	return slog.New(handler)
}
