package marketplace

import (
	"reverse-watch/internal/domain/service"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	keySvc         service.KeyService
	marketplaceSvc service.MarketplaceService
}

func NewMarketplaceHandler(keySvc service.KeyService, marketplaceSvc service.MarketplaceService) *Handler {
	return &Handler{
		keySvc:         keySvc,
		marketplaceSvc: marketplaceSvc,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/", h.createMarketplace)
	r.Patch("/", h.patchMarketplace)
	r.Delete("/{slug}", h.deleteMarketplace)
}
