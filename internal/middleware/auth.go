package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

type AuthMode string

const (
	AuthNone   AuthMode = "none"
	AuthAPIKey AuthMode = "apikey"
	AuthBearer AuthMode = "bearer"
)

type AuthConfig struct {
	Mode        AuthMode
	APIKey      string
	BearerToken string
	SkipPaths   []string
}

type authErr struct {
	Error string `json:"error"`
}

func AuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	// normalize skip path set
	skip := make(map[string]struct{}, len(cfg.SkipPaths))
	for _, p := range cfg.SkipPaths {
		skip[p] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		if cfg.Mode == AuthNone {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			switch cfg.Mode {
			case AuthAPIKey:
				// Header: X-API-Key: <key>
				got := r.Header.Get("X-API-Key")
				if constantTimeEq(got, cfg.APIKey) {
					next.ServeHTTP(w, r)
					return
				}
				unauthorized(w, `ApiKey realm="tasks", header="X-API-Key"`)
				return

			case AuthBearer:
				// Header: Authorization: Bearer <token>
				authz := r.Header.Get("Authorization")
				if token := strings.TrimPrefix(authz, "Bearer "); token != authz && constantTimeEq(strings.TrimSpace(token), cfg.BearerToken) {
					next.ServeHTTP(w, r)
					return
				}
				unauthorized(w, `Bearer realm="tasks"`)
				return

			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func constantTimeEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func unauthorized(w http.ResponseWriter, challenge string) {
	w.Header().Set("Content-Type", "application/json")
	if challenge != "" {
		w.Header().Set("WWW-Authenticate", challenge)
	}
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(authErr{Error: "unauthorized"})
}
