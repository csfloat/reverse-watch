package admin

import (
	"reverse-watch/api/v1/admin/marketplace"
	"reverse-watch/domain/models"
	"reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)
	r.Use(middleware.RequirePermissions(models.PermissionAdmin))
	
	r.Mount("/marketplace", marketplace.Router())
	return r
}
