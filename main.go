package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/s1natex/tasks-api-GO/internal/middleware"
	"github.com/s1natex/tasks-api-GO/internal/tasks"
)

// main is the entry point of the application
// It sets up logging, the router with middleware, and starts the HTTP server
// It also handles graceful shutdown on OS signals
func main() {
	logger := newLoggerFromEnv()
	slog.SetDefault(logger)

	repo := tasks.NewInMemoryRepo()
	app := newRouter(repo, logger)
	health := healthRouter()

	appSrv := &http.Server{
		Addr:              ":8080",
		Handler:           app,
		ReadHeaderTimeout: 5 * time.Second,
	}
	healthSrv := &http.Server{
		Addr:              ":8081",
		Handler:           health,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		logger.Info("server_listen", slog.String("addr", appSrv.Addr))
		if err := appSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server_error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	go func() {
		logger.Info("health_listen", slog.String("addr", healthSrv.Addr))
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("health_error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	<-ctx.Done()
	stop()

	logger.Info("shutdown_begin")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := appSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown_app_error", slog.String("error", err.Error()))
	}

	if err := healthSrv.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown_health_error", slog.String("error", err.Error()))
	}

	logger.Info("shutdown_complete")
}

// newRouter creates a chi.Mux with all application routes and middleware
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

	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		middleware.MetricsHandler().ServeHTTP(w, r)
	})

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	tasks.RegisterRoutes(r, repo)
	return r
}

// healthRouter returns a simple router with just the /health endpoint
func healthRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
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
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	return slog.New(handler)
}
