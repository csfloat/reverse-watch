package admin

import (
	"reverse-watch/api/v1/admin/keys"
	"reverse-watch/api/v1/admin/marketplace"
	"reverse-watch/api/v1/admin/reversals"
	"reverse-watch/api/v1/admin/users"
	"reverse-watch/domain/models"
	"reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.AuthMiddleware)
	r.Use(middleware.RequirePermissions(models.PermissionAdmin))

	r.Mount("/marketplace", marketplace.Router())
	r.Mount("/keys", keys.Router())
	r.Mount("/reversals", reversals.Router())
	r.Mount("/users", users.Router())
	return r
}
