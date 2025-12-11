package keys

import (
	"reverse-watch/middleware"
	"reverse-watch/services/private"

	"github.com/go-chi/chi/v5"
)

func Router(privateSvc *private.Service) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.PrivateServiceMiddleware(privateSvc))

	r.Post("/", adminCreateKeyHandler)
	return r
}
