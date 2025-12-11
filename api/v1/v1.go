package v1

import (
	"reverse-watch/api/v1/admin"
	"reverse-watch/api/v1/marketplace"
	"reverse-watch/services/private"
	"reverse-watch/services/public"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service, publicSvc *public.Service) chi.Router {
	r := chi.NewRouter()

	r.Mount("/marketplace", marketplace.Router(privateSvc))
	r.Mount("/admin", admin.Router(privateSvc, publicSvc))

	return r
}
