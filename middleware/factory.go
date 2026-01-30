package middleware

import (
	"context"
	"net/http"

	"reverse-watch/domain/repository"
)

const FactoryContextKey ContextKey = "factory"

func Factory(factory repository.Factory) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), FactoryContextKey, factory)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
