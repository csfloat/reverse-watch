package api

import (
	"reverse-watch/api/v1"
	"reverse-watch/services/private"
	"reverse-watch/services/public"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service, publicSvc *public.Service) chi.Router {
	r := chi.NewRouter()

	r.Mount("/v1", v1.Router(privateSvc, publicSvc))

	return r
}
