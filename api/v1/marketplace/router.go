package marketplace

import (
	"reverse-watch/domain/models"
	rwmiddleware "reverse-watch/middleware"
	"reverse-watch/ratelimit"
	"time"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(rwmiddleware.AuthMiddleware)
	r.Use(rwmiddleware.RequirePermissions(models.PermissionManage))

	r.Route("/keys", func(r chi.Router) {
		r.With(ratelimit.ThrottleByAPIKey(time.Minute, 10)).Get("/", listKeys)
		r.With(ratelimit.ThrottleByAPIKey(15*time.Minute, 2)).Post("/", createKey)
		r.With(ratelimit.ThrottleByAPIKey(15*time.Minute, 2)).Delete("/{id}", deleteKey)
	})
	return r
}
