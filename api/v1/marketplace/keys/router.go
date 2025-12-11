package keys

import (
	"reverse-watch/middleware"
	"reverse-watch/services/private"
	"reverse-watch/types"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.PrivateServiceMiddleware(privateSvc), middleware.RequirePermission(types.PermissionManage))

	r.Route("/", func(r chi.Router) {
		r.Get("/", listKeysHandler)
		r.Post("/", createKeyHandler)
		r.Delete("/{id}", deleteKeyHandler)
	})
	return r
}
