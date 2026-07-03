package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ekhrunov/messenger/server/internal/service"
)

func RequireAuth(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, err := auth.AuthenticateRequest(r)
			if err != nil {
				writeUnauthorized(w, err)
				return
			}

			ctx := r.Context()
			ctx = contextWithTokenUser(ctx, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func contextWithTokenUser(ctx context.Context, user service.TokenUser) context.Context {
	return context.WithValue(ctx, service.TokenUserContextKey, user)
}

func writeUnauthorized(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
