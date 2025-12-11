package keys

import (
	"encoding/json"
	"net/http"

	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
	"reverse-watch/services/private"
	"reverse-watch/types"

	"github.com/go-chi/chi/v5"
)

func createKeyHandler(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	var req struct {
		Scope *types.Scope `json:"scope"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.Scope == nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := privateSvc.NewKey(key.MarketplaceSlug, *req.Scope)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	render.JSON(w, r, rawKey)
}

func listKeysHandler(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	keysList, err := privateSvc.ListKeys(&private.ListKeyOptions{
		MarketplaceSlug: key.MarketplaceSlug,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list keys")
		return
	}

	render.JSON(w, r, keysList)
}

func deleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	idStr := chi.URLParam(r, "id")
	id, err := types.ToSnowflake(idStr)
	if err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	keyToDelete, err := privateSvc.GetKey(id)
	if err != nil {
		render.Error(w, r, &errors.NotFound)
		return
	}

	// Ensure API consumer owns the key being deleted
	if key.MarketplaceSlug != keyToDelete.MarketplaceSlug {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if key.Scope < keyToDelete.Scope {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if err := privateSvc.DeleteKey(id); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
