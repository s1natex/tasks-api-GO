package middleware

import (
	"encoding/json"
	"net/http"
	"strconv"

	"golang.org/x/time/rate"
)

type rateErr struct {
	Error string `json:"error"`
}

func RateLimitMiddleware(l *rate.Limiter) func(http.Handler) http.Handler {
	if l == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if l.Allow() {
				next.ServeHTTP(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
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
