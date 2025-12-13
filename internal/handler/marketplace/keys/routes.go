package keys

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keySvc service.KeyService
}

func NewKeyHandler(keySvc service.KeyService) *Handler {
	return &Handler{
		keySvc: keySvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.listKeysHandler)
	r.Post("/", h.createKeyHandler)
	r.Delete("/{id}", h.deleteKeyHandler)
}
