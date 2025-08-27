package middleware

import (
	"log/slog"
	"net/http"
	"time"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// statusWriter wraps http.ResponseWriter so we can capture status code & byte size
type statusWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		// If Write is called without WriteHeader, status defaults to 200
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// RequestLogger logs method, path, status, duration, size, ip, user-agent, and request_id.
// {"time":"...","level":"INFO","msg":"http_request","req_id":"7b3a...","method":"GET","path":"/tasks","status":200,"duration_ms":1.23,"size":123,"ip":"127.0.0.1:54321","ua":"curl/8.6.0"}
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}

			next.ServeHTTP(sw, r)

			dur := time.Since(start)
			ip := clientIP(r)
			reqID := chimw.GetReqID(r.Context())
			logger.Info("http_request",
				slog.String("req_id", reqID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Float64("duration_ms", float64(dur.Microseconds())/1000.0),
				slog.Int("size", sw.bytes),
				slog.String("ip", ip),
				slog.String("ua", r.UserAgent()),
			)
		})
	}
}

func clientIP(r *http.Request) string {
	return r.RemoteAddr
}
