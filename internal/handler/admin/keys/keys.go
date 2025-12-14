package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) adminCreateKeyHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MarketplaceSlug string             `json:"marketplace_slug"`
		Permissions     models.Permissions `json:"permissions"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || req.Permissions == 0 {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := h.keySvc.CreateKey(req.MarketplaceSlug, req.Permissions)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}

func (h *Handler) adminDeleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	if err := h.keySvc.DeleteKey(snowflake); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
