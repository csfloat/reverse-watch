package marketplace

import (
	"reverse-watch/api/v1/marketplace/keys"
	"reverse-watch/middleware"
	"reverse-watch/services/private"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.Middleware(privateSvc))

	r.Mount("/keys", keys.Router(privateSvc))
	return r
}
