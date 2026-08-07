package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type AuthMiddleware struct {
	jwt    *JWTValidator
	apiKey *APIKeyValidator
}

func NewAuthMiddleware(jwtSecret, redisHost, redisPort, redisPassword string) *AuthMiddleware {
	return &AuthMiddleware{
		jwt:    NewJWTValidator(jwtSecret),
		apiKey: NewAPIKeyValidator(redisHost, redisPort, redisPassword),
	}
}

func (a *AuthMiddleware) Protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		apiKey := r.Header.Get("X-API-Key")

		if apiKey != "" {
			userID, err := a.apiKey.Validate(apiKey)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid API key")
				return
			}
			r.Header.Set("X-User-ID", userID)
			r.Header.Set("X-Auth-Method", "api-key")
			next.ServeHTTP(w, r)
			return
		}

		if authHeader == "" {
			writeAuthError(w, http.StatusUnauthorized, "missing authorization token")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeAuthError(w, http.StatusUnauthorized, "invalid authorization format, use: Bearer <token>")
			return
		}

		claims, err := a.jwt.Validate(parts[1])
		if err != nil {
			writeAuthError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-User-Email", claims.Email)
		r.Header.Set("X-User-Role", claims.Role)
		r.Header.Set("X-Auth-Method", "jwt")

		next.ServeHTTP(w, r)
	})
}

func writeAuthError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":  message,
		"status": status,
	})
}
