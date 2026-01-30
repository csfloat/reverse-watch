package marketplace

import (
	"reverse-watch/api/v1/marketplace/keys"
	rwmiddleware "reverse-watch/middleware"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Use(rwmiddleware.AuthMiddleware)
	r.Mount("/keys", keys.Router())
	return r
}
