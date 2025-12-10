package middleware

import (
	"net/http"

	"reverse-watch/errors"
	"reverse-watch/render"
	"reverse-watch/types"
)

func RequirePermission(permission types.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			key, ok := r.Context().Value(KeyContextKey).(*types.Key)
			if !ok {
				render.Error(w, r, &errors.NotAuthorized)
				return
			}

			if !key.Scope.HasPermission(permission) {
				render.Error(w, r, &errors.NotAuthorized)
				return
			}
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
