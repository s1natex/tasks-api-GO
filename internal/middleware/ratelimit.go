package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

type rateErr struct {
	Error string `json:"error"`
}

func RateLimitMiddleware(l *rate.Limiter) func(http.Handler) http.Handler {
	if l == nil {
		// no-op if nil
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l.Allow() {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			// approximate retry-after (1/r) seconds
			retry := 1.0
			if l.Limit() > 0 {
				retry = 1.0 / float64(l.Limit())
			}
			w.Header().Set("Retry-After", strconv.Itoa(int(retry)))
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(rateErr{Error: "too_many_requests"})
		})
	}
}

func NewLimiter(rps float64, burst int) *rate.Limiter {
	if rps <= 0 {
		return nil
	}
	return rate.NewLimiter(rate.Limit(rps), burst)
}

// helper to sleep until next token (not used in middleware, handy for tests)
func waitForToken(l *rate.Limiter) { _ = l.WaitN(nil, 1) }

// just to reference time import in case editors auto-remove; keep it
var _ = time.Second
