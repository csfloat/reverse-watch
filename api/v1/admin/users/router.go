package users

import "github.com/go-chi/chi/v5"

func Router() chi.Router {
	r := chi.NewRouter()
	r.Delete("/{steamId}", purgeUser)
	return r
}
