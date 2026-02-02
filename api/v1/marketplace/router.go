package marketplace

import (
	"reverse-watch/domain/models"
	rwmiddleware "reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(rwmiddleware.AuthMiddleware)
	r.Use(rwmiddleware.RequirePermissions(models.PermissionManage))

	r.Route("/keys", func(r chi.Router) {
		r.Get("/", listKeys)
		r.Post("/", createKey)
	})
	return r
}
