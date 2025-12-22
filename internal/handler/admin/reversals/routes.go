package reversals

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	reversalSvc   service.ReversalService
	adminAuditSvc service.AdminAuditService
}

func NewReversalHandler(reversalSvc service.ReversalService, adminAuditSvc service.AdminAuditService) *Handler {
	return &Handler{
		reversalSvc:   reversalSvc,
		adminAuditSvc: adminAuditSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Patch("/{id}", h.patchReversalHandler)
	r.Delete("/{id}", h.deleteReversalHandler)
}
