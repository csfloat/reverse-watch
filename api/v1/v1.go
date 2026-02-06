package v1

import (
	"reverse-watch/api/v1/marketplace"
	"reverse-watch/api/v1/reversals"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/marketplace", marketplace.Router())
	r.Mount("/reversals", reversals.Router())
	return r
}
