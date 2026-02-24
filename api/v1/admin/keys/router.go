package keys

import (
	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()

	r.Post("/", createKey)
	r.Delete("/{id}", deleteKey)
	return r
}
