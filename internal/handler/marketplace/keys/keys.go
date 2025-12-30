package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/middleware"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) createKeyHandler(w http.ResponseWriter, r *http.Request) {
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

	rawKey, err := h.keySvc.CreateKey(key.MarketplaceSlug, req.Permissions)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}

func (h *Handler) listKeysHandler(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	keysList, err := h.keySvc.ListKeys(&dto.KeyListOptions{
		MarketplaceSlug: &key.MarketplaceSlug,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list keys")
		return
	}

	render.JSON(w, r, keysList)
}

func (h *Handler) deleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	id := chi.URLParam(r, "id")

	keyToDelete, err := h.keySvc.GetKey(id)
	if err != nil {
		render.Error(w, r, &errors.NotFound)
		return
	}

	// Ensure API consumer owns the key being deleted
	if key.MarketplaceSlug != keyToDelete.MarketplaceSlug {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if err := h.keySvc.DeleteKey(id); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
