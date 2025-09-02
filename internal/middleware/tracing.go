package middleware

import (
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

func TracingMiddleware(next http.Handler) http.Handler {
	tr := otel.Tracer("http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		reqID := chimw.GetReqID(ctx)

		sw := &statusWriter{ResponseWriter: w}
		start := time.Now()

		ctx, span := tr.Start(ctx, r.Method+" "+r.URL.Path)
		defer span.End()

		r = r.WithContext(ctx)
		next.ServeHTTP(sw, r)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		dur := time.Since(start).Milliseconds()

		span.SetAttributes(
			attribute.String("http.method", r.Method),
			attribute.String("http.target", r.URL.Path),
			attribute.Int("http.status_code", sw.status),
			attribute.String("request.id", reqID),
			attribute.Int64("http.duration_ms", dur),
		)

		if sc := span.SpanContext(); sc.IsValid() {
			w.Header().Set("Trace-Id", sc.TraceID().String())
		}
	})
}
