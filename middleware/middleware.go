package middleware

import (
	"context"
	"net/http"
	"strings"

	"reverse-watch/errors"
	"reverse-watch/render"
	"reverse-watch/services/private"
)

type ContextKey string

const KeyContextKey ContextKey = "key"

func Middleware(privateSvc *private.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			secretKey := strings.TrimPrefix(authHeader, "Bearer ")

			key, err := privateSvc.Validate(secretKey)
			if err != nil {
				render.Error(w, r, &errors.InvalidApiKey)
				return
			}

			r = r.WithContext(context.WithValue(r.Context(), KeyContextKey, key))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
