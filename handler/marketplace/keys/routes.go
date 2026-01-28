package keys

import (
	"reverse-watch/domain/repository"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keyRepo repository.KeyRepository
}

func NewKeyHandler(keyRepo repository.KeyRepository) *Handler {
	return &Handler{
		keyRepo: keyRepo,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.createKey)
}
