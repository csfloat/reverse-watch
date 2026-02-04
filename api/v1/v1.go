package v1

import (
	"reverse-watch/api/v1/marketplace"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/marketplace", marketplace.Router())
	return r
}
