package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"golang.org/x/time/rate"

	appmw "github.com/s1natex/tasks-api-GO/internal/middleware"
)

func TestRateLimit(t *testing.T) {
	lim := rate.NewLimiter(1, 1) // 1 rps, burst 1
	r := chi.NewRouter()
	r.Use(appmw.RateLimitMiddleware(lim))
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// first allowed
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// second immediately should be 429
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/ping", nil)
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", rec.Code)
	}
}
