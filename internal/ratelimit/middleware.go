package ratelimit

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Config struct {
	Limit  int
	Window time.Duration
}

type Middleware struct {
	limiter *Limiter
	config  Config
}

func NewMiddleware(limiter *Limiter, limit int, windowSeconds int) *Middleware {
	return &Middleware{
		limiter: limiter,
		config: Config{
			Limit:  limit,
			Window: time.Duration(windowSeconds) * time.Second,
		},
	}
}

func (m *Middleware) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("rl:%s:%s", getClientIP(r), r.URL.Path)

		result, err := m.limiter.Allow(r.Context(), key, m.config.Limit, m.config.Window)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", m.config.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))
		w.Header().Set("X-RateLimit-Window", fmt.Sprintf("%ds", int(m.config.Window.Seconds())))

		if !result.Allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", result.RetryAfter))
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":       "rate limit exceeded",
				"limit":       m.config.Limit,
				"window":      fmt.Sprintf("%ds", int(m.config.Window.Seconds())),
				"retry_after": result.RetryAfter,
				"status":      429,
			})
			return
		}

		next.ServeHTTP(w, r)
	})
}

func getClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.Split(ip, ",")[0]
	}
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx != -1 {
		return r.RemoteAddr[:idx]
	}
	return r.RemoteAddr
}
