package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/middleware"
	"reverse-watch/internal/render"

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
	r.With(middleware.RequirePermissions(models.PermissionWrite)).Post("/", h.createReversalHandler)
}
