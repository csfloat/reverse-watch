package middleware

import (
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"
)

func RequirePermissions(permissions ...models.Permissions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			key, ok := r.Context().Value(KeyContextKey).(*models.Key)
			if !ok {
				render.Error(w, r, &errors.NotAuthorized)
				return
			}

			if !key.HasPermissions(permissions...) {
				render.Error(w, r, &errors.NotAuthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
