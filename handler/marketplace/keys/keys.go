package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/models"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
)

func (h *Handler) createKey(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	var req struct {
		Permissions models.Permissions `json:"permissions"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.Permissions == 0 {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := h.keyRepo.Create(key.MarketplaceSlug, req.Permissions)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}
