package admin

import (
	"reverse-watch/api/v1/admin/keys"
	"reverse-watch/api/v1/admin/marketplace"
	"reverse-watch/middleware"
	"reverse-watch/services/private"
	"reverse-watch/services/public"
	"reverse-watch/types"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service, publicSvc *public.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Middleware(privateSvc))
	r.Use(middleware.RequirePermission(types.PermissionAdmin))

	r.Mount("/marketplace", marketplace.Router(privateSvc))
	r.Mount("/keys", keys.Router(privateSvc))
	return r
}
