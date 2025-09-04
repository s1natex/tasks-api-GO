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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"strconv"
)

//go:embed openapi/openapi.json
var openapiSpec []byte

//go:embed openapi/swagger.html
var swaggerHTML string

func main() {
	logger := newLoggerFromEnv()
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("fatal", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	shutdown, err := initTracer()
	if err != nil {
		return err
	}
	defer func() { _ = shutdown(context.Background()) }()

	dbPath := envDefault("DB_PATH", "data/tasks.db")
	dsn, err := tasks.SQLiteFileDSN(dbPath)
	if err != nil {
		return err
	}
	sqliteRepo, err := tasks.NewSQLiteRepo(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = sqliteRepo.Close() }()
	if err := sqliteRepo.ApplyMigrations(context.Background()); err != nil {
		return err
	}

	app := newRouter(sqliteRepo, logger)
	health := healthRouter()

	appSrv := &http.Server{Addr: ":8080", Handler: app, ReadHeaderTimeout: 5 * time.Second}
	healthSrv := &http.Server{Addr: ":8081", Handler: health, ReadHeaderTimeout: 2 * time.Second}

	errCh := make(chan error, 2)

	go func() {
		logger.Info("server_listen", slog.String("addr", appSrv.Addr))
		if err := appSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	go func() {
		logger.Info("health_listen", slog.String("addr", healthSrv.Addr))
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		// graceful shutdown below
	case err := <-errCh:
		logger.Error("server_error", slog.String("error", err.Error()))
	}

	logger.Info("shutdown_begin")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = appSrv.Shutdown(shutdownCtx)
	_ = healthSrv.Shutdown(context.Background())
	logger.Info("shutdown_complete")
	return nil
}

func newRouter(repo tasks.Repository, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(15 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key"},
		ExposedHeaders:   []string{"Link", "X-Request-ID", "Trace-Id"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	authCfg := middleware.AuthConfig{
		Mode:        AuthModeFromEnv(),
		APIKey:      strings.TrimSpace(os.Getenv("API_KEY")),
		BearerToken: strings.TrimSpace(os.Getenv("BEARER_TOKEN")),
		SkipPaths:   []string{"/health", "/openapi.json", "/docs", "/metrics"},
	}
	r.Use(middleware.AuthMiddleware(authCfg))

	rps := floatFromEnv("RATE_LIMIT_RPS", 0)
	burst := intFromEnv("RATE_LIMIT_BURST", 0)
	if rps > 0 && burst == 0 {
		burst = int(rps * 2)
	}
	r.Use(middleware.RateLimitMiddleware(middleware.NewLimiter(rps, burst)))

	r.Use(middleware.TracingMiddleware)
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
		_, _ = w.Write([]byte(`{"status":"ok"}`))
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

func initTracer() (func(context.Context) error, error) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	res, _ := sdkresource.Merge(
		sdkresource.Default(),
		sdkresource.NewWithAttributes(
			"",
			attribute.String("service.name", "tasks-api-GO"),
		),
	)

	var exp sdktrace.SpanExporter
	if ep := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")); ep != "" {
		client := otlptracehttp.NewClient(otlptracehttp.WithEndpoint(ep), otlptracehttp.WithInsecure())
		otlpExp, err := otlptrace.New(context.Background(), client)
		if err != nil {
			return nil, err
		}
		exp = otlpExp
	} else {
		stdExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
		exp = stdExp
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func AuthModeFromEnv() middleware.AuthMode {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("AUTH_MODE"))) {
	case "apikey":
		return middleware.AuthAPIKey
	case "bearer":
		return middleware.AuthBearer
	default:
		return middleware.AuthNone
	}
}

func floatFromEnv(k string, def float64) float64 {
	if s := strings.TrimSpace(os.Getenv(k)); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return def
}

func intFromEnv(k string, def int) int {
	if s := strings.TrimSpace(os.Getenv(k)); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			return v
		}
	}
	return def
}
