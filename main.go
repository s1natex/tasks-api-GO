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

func main() {
	logger := newLoggerFromEnv()
	slog.SetDefault(logger) // for third-party packages that use slog

	repo := tasks.NewInMemoryRepo()
	r := newRouter(repo, logger)

	logger.Info("server_listen", slog.String("addr", ":8080"))
	if err := http.ListenAndServe(":8080", r); err != nil {
		logger.Error("server_error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// newRouter wires the health endpoint, task routes, and middleware stack
func newRouter(repo tasks.Repository, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	// ---- Middleware stack (order matters a bit) ----
	// RequestID first so downstream can include it (logger, errors, etc.)
	r.Use(chimw.RequestID)

	// Panic recovery: never crash the server; returns 500 on panics
	r.Use(chimw.Recoverer)

	// Timeouts: cancel handlers that exceed this duration
	r.Use(chimw.Timeout(15 * time.Second))

	// CORS
	// Allow all origins, methods, and headers for demo purposes
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"}, // e.g., []string{"https://your-frontend.example"}
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300, // 5 minutes
	}))

	// Our structured request logger (now includes req_id).
	r.Use(middleware.RequestLogger(logger))

	// ---- Routes ----

	// health
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// tasks routes (POST /tasks, GET /tasks)
	tasks.RegisterRoutes(r, repo)

	return r
}

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
