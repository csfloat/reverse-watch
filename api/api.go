package api

import (
	v1 "reverse-watch/api/v1"

	"github.com/go-chi/chi/v5"
)

func Router(steamWebAPIKey string) chi.Router {
	r := chi.NewRouter()
	r.Mount("/v1", v1.Router(steamWebAPIKey))
	return r
}
