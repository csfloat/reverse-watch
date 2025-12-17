package reversals

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keySvc      service.KeyService
	reversalSvc service.ReversalService
}

func NewReversalHandler(keySvc service.KeyService, reversalSvc service.ReversalService) *Handler {
	return &Handler{
		keySvc:      keySvc,
		reversalSvc: reversalSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Use(middleware.RequirePermissions(models.PermissionWrite))

	r.Post("/", h.createReversalsHandler)
	r.Delete("/{id}", h.expungeReversalHandler)
}
