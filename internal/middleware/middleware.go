package middleware

import (
	"context"
	"net/http"
	"strings"

	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"
)

type ContextKey string

const KeyContextKey ContextKey = "key"

func Middleware(keySvc service.KeyService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			secretKey := strings.TrimPrefix(authHeader, "Bearer ")

			key, err := keySvc.ValidateKey(secretKey)
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
