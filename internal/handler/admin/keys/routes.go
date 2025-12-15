package keys

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keySvc        service.KeyService
	adminAuditSvc service.AdminAuditService
}

func NewKeyHandler(keySvc service.KeyService, adminAuditSvc service.AdminAuditService) *Handler {
	return &Handler{
		keySvc:        keySvc,
		adminAuditSvc: adminAuditSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.adminCreateKeyHandler)
	r.Delete("/", h.adminDeleteKeyHandler)
}
