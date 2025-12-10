package marketplace

import (
	"reverse-watch/middleware"
	"reverse-watch/services/private"
	"reverse-watch/types"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Middleware(privateSvc))

	r.With(middleware.PrivateServiceMiddleware(privateSvc)).
		Route("/keys", func(r chi.Router) {
			r.Use(middleware.RequirePermission(types.PermissionManage))

			r.Get("/", listKeysHandler)
			r.Post("/", createKeyHandler)
			r.Delete("/{id}", deleteKeyHandler)
		})
	return r
}
