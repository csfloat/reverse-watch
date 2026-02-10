package reversals

import (
	"reverse-watch/domain/models"
	"reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)

	r.With(middleware.RequirePermissions(models.PermissionWrite)).Post("/", createReversals)
	r.With(middleware.RequirePermissions(models.PermissionExport)).Get("/", listReversalsHandler)
	return r
}
