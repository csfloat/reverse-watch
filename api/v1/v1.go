package v1

import (
	"reverse-watch/api/v1/admin"
	"reverse-watch/api/v1/marketplace"
	"reverse-watch/api/v1/reversals"
	"reverse-watch/api/v1/users"

	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()
	r.Mount("/marketplace", marketplace.Router())
	r.Mount("/reversals", reversals.Router())
	r.Mount("/users", users.Router())
	r.Mount("/admin", admin.Router())
	return r
}
