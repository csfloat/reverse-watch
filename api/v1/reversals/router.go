package reversals

import (
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/middleware"
	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()

	r.With(ratelimit.ThrottleByIP(time.Minute, 30)).Get("/recent", listRecentHandler)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.With(
			middleware.RequirePermissions(models.PermissionWrite),
			ratelimit.ThrottleByAPIKey(time.Hour, 2_000),
		).Post("/", createReversals)

		r.With(
			middleware.RequirePermissions(models.PermissionDelete),
			ratelimit.ThrottleByAPIKey(time.Hour, 2_000),
		).Delete("/{id}", expungeReversal)

		r.Route("/", func(r chi.Router) {
			r.Use(middleware.RequirePermissions(models.PermissionExport))

			r.With(ratelimit.ThrottleByAPIKey(time.Minute, 300)).Get("/", listReversalsHandler)
			r.With(ratelimit.ThrottleByAPIKey(time.Minute, 60)).Get("/export", exportReversals)
		})
	})

	return r
}
