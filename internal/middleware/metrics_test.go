package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	appmw "github.com/s1natex/tasks-api-GO/internal/middleware"
)

func TestMetricsCounterIncrements(t *testing.T) {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.Recoverer)
	r.Use(appmw.MetricsMiddleware)

	// simple handler
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	})

	// fire one request to increment counter
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// scrape /metrics
	mreq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	mrec := httptest.NewRecorder()
	appmw.MetricsHandler().ServeHTTP(mrec, mreq)
	if mrec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /metrics, got %d", mrec.Code)
	}
	body := mrec.Body.String()

	// assert counter line exists
	want := `http_requests_total{method="GET",path="/ping",status="200"} 1`
	if !strings.Contains(body, want) {
		t.Fatalf("expected metrics to contain %q\nfull body:\n%s", want, body)
	}
}
