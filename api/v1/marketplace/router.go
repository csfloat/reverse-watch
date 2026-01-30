package marketplace

import (
	"reverse-watch/api/v1/marketplace/keys"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/keys", keys.Router())
	return r
}
