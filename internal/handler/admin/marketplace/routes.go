package marketplace

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keySvc         service.KeyService
	marketplaceSvc service.MarketplaceService
	adminAuditSvc  service.AdminAuditService
}

func NewMarketplaceHandler(keySvc service.KeyService, marketplaceSvc service.MarketplaceService, adminAuditSvc service.AdminAuditService) *Handler {
	return &Handler{
		keySvc:         keySvc,
		marketplaceSvc: marketplaceSvc,
		adminAuditSvc:  adminAuditSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.createMarketplace)
	r.Patch("/{slug}", h.patchMarketplace)
	r.Delete("/{slug}", h.deleteMarketplace)
}
