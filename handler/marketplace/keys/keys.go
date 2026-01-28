package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
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

func (h *Handler) listKeys(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	keysList, err := h.keyRepo.List(&dto.KeyListOptions{
		MarketplaceSlug: &key.MarketplaceSlug,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list keys")
		return
	}

	render.JSON(w, r, keysList)
}

func (h *Handler) deleteKey(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	id := chi.URLParam(r, "id")

	keyToDelete, err := h.keyRepo.Read(id)
	if err != nil {
		render.Error(w, r, &errors.NotFound)
		return
	}

	// Ensure API consumer owns the key being deleted
	if key.MarketplaceSlug != keyToDelete.MarketplaceSlug {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if err := h.keyRepo.Delete(id); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
