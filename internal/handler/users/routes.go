package users

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	reversalSvc service.ReversalService
}

func NewUsersHandler(reversalSvc service.ReversalService) *Handler {
	return &Handler{
		reversalSvc: reversalSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/{steam_id}", h.getReversalStatus)
}
