package middleware

import (
	"context"
	"net/http"

	"reverse-watch/services/private"
	"reverse-watch/services/public"
)

const PrivateServiceContextKey ContextKey = "privateService"

func PrivateServiceMiddleware(privateSvc *private.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), PrivateServiceContextKey, privateSvc))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}

const PublicServiceContextKey ContextKey = "publicService"

func PublicServiceMiddleware(publicSvc *public.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(context.WithValue(r.Context(), PublicServiceContextKey, publicSvc))
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	}
}
