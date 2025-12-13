package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"
)

func (h *Handler) adminCreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MarketplaceSlug string       `json:"marketplace_slug"`
		Scope           models.Scope `json:"scope"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || !req.Scope.IsValid() {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := h.keySvc.CreateKey(req.MarketplaceSlug, req.Scope)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}
