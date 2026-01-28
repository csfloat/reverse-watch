package middleware

import (
	"context"
	"net/http"
	"strings"

	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/render"
)

type ContextKey string

const KeyContextKey ContextKey = "key"

func Middleware(keyRepo repository.KeyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			secretKey := strings.TrimPrefix(authHeader, "Bearer ")

			key, err := keyRepo.ValidateKey(secretKey)
			if err != nil {
				render.Error(w, r, &errors.InvalidApiKey)
				return
			}

			ctx := context.WithValue(r.Context(), KeyContextKey, key)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
