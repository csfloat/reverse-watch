package middleware

import (
	"context"
	"net/http"

	steamservice "reverse-watch/service/steam"
)

const SteamServiceContextKey ContextKey = "steamService"

func SteamServiceMiddleware(svc *steamservice.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), SteamServiceContextKey, svc)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}
