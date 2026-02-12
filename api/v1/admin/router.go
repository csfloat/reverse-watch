package admin

import (
	"reverse-watch/api/v1/admin/marketplace"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/marketplace", marketplace.Router())
	return r
}
