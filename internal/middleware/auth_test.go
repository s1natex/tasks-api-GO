package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	appmw "github.com/s1natex/tasks-api-GO/internal/middleware"
)

func TestAuth_APIKey(t *testing.T) {
	r := chi.NewRouter()
	r.Use(appmw.AuthMiddleware(appmw.AuthConfig{
		Mode:      appmw.AuthAPIKey,
		APIKey:    "secret123",
		SkipPaths: []string{"/health"},
	}))
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	r.Get("/tasks", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tasks", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("X-API-Key", "secret123")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with key, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/health", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("skip path should be open, got %d", rec.Code)
	}
}

func TestAuth_Bearer(t *testing.T) {
	r := chi.NewRouter()
	r.Use(appmw.AuthMiddleware(appmw.AuthConfig{
		Mode:        appmw.AuthBearer,
		BearerToken: "tok_abc",
	}))
	r.Get("/tasks", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tasks", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("Authorization", "Bearer tok_abc")
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with bearer, got %d", rec.Code)
	}
}
