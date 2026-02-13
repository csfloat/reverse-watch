package marketplace

import (
	"reverse-watch/domain/models"
	"reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)
	r.With(middleware.RequirePermissions(models.PermissionAdmin)).Post("/", onboardMarketplace)
	return r
}
