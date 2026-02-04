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

func parseBearerToken(authHeader string) (string, bool) {
	prefix := "bearer "
	if len(authHeader) < len(prefix) {
		return "", false
	}
	if !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return "", false
	}
	token := authHeader[len(prefix):]
	if token == "" {
		return "", false
	}
	return token, true
}

func AuthMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		factory, ok := r.Context().Value(FactoryContextKey).(repository.Factory)
		if !ok {
			render.Error(w, r, &errors.InternalServerError)
			return
		}

		authHeader := r.Header.Get("Authorization")
		secretKey, ok := parseBearerToken(authHeader)
		if !ok {
			render.Error(w, r, &errors.InvalidApiKey)
			return
		}

		key, err := factory.Key().ValidateKey(secretKey)
		if err != nil {
			render.Error(w, r, &errors.InvalidApiKey)
			return
		}

		ctx := context.WithValue(r.Context(), KeyContextKey, key)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}
