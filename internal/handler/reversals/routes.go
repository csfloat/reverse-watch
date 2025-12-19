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
	r.With(middleware.RequirePermissions(models.PermissionWrite)).Post("/", h.createReversalsHandler)
	r.With(middleware.RequirePermissions(models.PermissionDelete)).Delete("/{id}", h.expungeReversalHandler)
	r.With(middleware.RequirePermissions(models.PermissionExport)).Route("/", func(r chi.Router) {
		r.Get("/", h.listReversalsHandler)
		r.Get("/export", h.exportReversalsHandler)
	})
}
