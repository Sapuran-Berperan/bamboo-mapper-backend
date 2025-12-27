package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/auth"
)

type contextKey string

const ClaimsKey contextKey = "claims"

// JWTAuth creates a middleware that validates JWT access tokens
func JWTAuth(jwtManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondUnauthorized(w, "Authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				respondUnauthorized(w, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]
			claims, err := jwtManager.ValidateAccessToken(tokenString)
			if err != nil {
				if err == auth.ErrExpiredToken {
					respondUnauthorized(w, "Token has expired")
					return
				}
				respondUnauthorized(w, "Invalid token")
				return
			}

			// Store claims in context
			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves the JWT claims from the request context
func GetClaims(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(ClaimsKey).(*auth.Claims)
	return claims, ok
}

// respondUnauthorized sends a 401 response with the standard format
func respondUnauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"meta":{"success":false,"message":"` + message + `"},"data":null}`))
}
