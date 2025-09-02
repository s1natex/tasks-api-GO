package main

import (
	"context"
	_ "embed"
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

//go:embed openapi/openapi.json
var openapiSpec []byte

//go:embed openapi/swagger.html
var swaggerHTML string

func main() {
	logger := newLoggerFromEnv()
	slog.SetDefault(logger)

	dbPath := envDefault("DB_PATH", "data/tasks.db")
	dsn, err := tasks.SQLiteFileDSN(dbPath)
	if err != nil {
		logger.Error("dsn_error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	sqliteRepo, err := tasks.NewSQLiteRepo(dsn)
	if err != nil {
		logger.Error("sqlite_open_error", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer sqliteRepo.Close()
	if err := sqliteRepo.ApplyMigrations(context.Background()); err != nil {
		logger.Error("sqlite_migrate_error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	app := newRouter(sqliteRepo, logger)
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
	_ = appSrv.Shutdown(shutdownCtx)
	_ = healthSrv.Shutdown(context.Background())

	logger.Info("shutdown_complete")
}

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

	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(openapiSpec)
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(swaggerHTML))
	})

	tasks.RegisterRoutes(r, repo)
	return r
}

func healthRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
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
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l})
	return slog.New(handler)
}

func envDefault(k, v string) string {
	if s := strings.TrimSpace(os.Getenv(k)); s != "" {
		return s
	}
	return v
}
