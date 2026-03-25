package api

import (
	v1 "reverse-watch/api/v1"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/v1", v1.Router())
	return r
}
