package reversals

import (
	"github.com/go-chi/chi/v5"
)

func Router() chi.Router {
	r := chi.NewRouter()

	r.Patch("/{id}", modifyReversal)
	r.Delete("/{id}", deleteReversal)
	return r
}
