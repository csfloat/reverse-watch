package marketplace

import (
	"reverse-watch/middleware"
	"reverse-watch/services/private"
	"reverse-watch/types"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service) chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequirePermission(types.PermissionAdmin))
	r.With(middleware.PrivateServiceMiddleware(privateSvc)).
		Route("/", func(r chi.Router) {
			r.Post("/", createMarketplace)
			r.Patch("/{slug}", patchMarketplace)
			r.Delete("/{slug}", deleteMarketplace)
		})
	return r
}
