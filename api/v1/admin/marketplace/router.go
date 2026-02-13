package marketplace

import (
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/middleware"
	"reverse-watch/ratelimit"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)
	r.Use(middleware.RequirePermissions(models.PermissionAdmin))
	r.Use(ratelimit.ThrottleByAPIKey(time.Hour, 2_000))

	r.Post("/", onboardMarketplace)
	r.Patch("/{slug}", updateMarketplace)
	return r
}
