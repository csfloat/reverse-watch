package marketplace

import (
	"time"

	"reverse-watch/domain/models"
	rwmiddleware "reverse-watch/middleware"
	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(rwmiddleware.AuthMiddleware)
	r.Use(rwmiddleware.RequirePermissions(models.PermissionManage))

	r.Route("/keys", func(r chi.Router) {
		r.Use(ratelimit.ThrottleByMarketplace(time.Minute, 100))

		r.Get("/", listKeys)
		r.Post("/", createKey)
		r.Delete("/{id}", deleteKey)
	})
	return r
}
