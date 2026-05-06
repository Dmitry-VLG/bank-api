package httpapi

import (
	"context"
	"net/http"
	"strings"

	"bank-api/internal/security"
)

type contextKey string

const userIDKey contextKey = "user_id"

func AuthMiddleware(tokens security.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			const prefix = "Bearer "
			if !strings.HasPrefix(header, prefix) {
				writeError(w, http.StatusUnauthorized, "invalid authorization header")
				return
			}

			userID, err := tokens.Parse(strings.TrimPrefix(header, prefix))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func userIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok && id > 0
}
