package marketplace

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
	keySvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	var req struct {
		Scope string `json:"scope"`
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	scopeEnum, err := keySvc.GetScopeEnum(req.Scope)
	if err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	rawKey, err := private.NewRawKey(key.MarketplaceSlug, scopeEnum.Scope)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	if err := keySvc.CreateKey(rawKey.ToKey()); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	render.JSON(w, r, struct {
		ID              types.Snowflake `json:"id"`
		SecretKey       string          `json:"secret_key"`
		MarketplaceSlug string          `json:"marketplace_slug"`
		Scope           string          `json:"scope"`
	}{
		ID:              rawKey.ID,
		SecretKey:       rawKey.SecretKey,
		MarketplaceSlug: rawKey.MarketplaceSlug,
		Scope:           rawKey.Scope.String(),
	})
}

func listKeysHandler(w http.ResponseWriter, r *http.Request) {
	keySvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	keysList, err := keySvc.ListKeys(&private.ListKeyOptions{
		MarketplaceSlug: key.MarketplaceSlug,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list keys")
		return
	}

	// Sanitize keys by removing the hash
	type sanitizedKey struct {
		ID              types.Snowflake `json:"id"`
		CreatedAt       uint64          `json:"created_at"`
		MarketplaceSlug string          `json:"marketplace_slug"`
		Scope           string          `json:"scope"`
	}

	sanitizedKeys := make([]*sanitizedKey, 0)
	for _, key := range keysList {
		sanitized := &sanitizedKey{
			ID:              key.ID,
			CreatedAt:       key.CreatedAt,
			MarketplaceSlug: key.MarketplaceSlug,
			Scope:           key.Scope.String(),
		}
		sanitizedKeys = append(sanitizedKeys, sanitized)
	}

	render.JSON(w, r, sanitizedKeys)
}

func deleteKeyHandler(w http.ResponseWriter, r *http.Request) {
	keySvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)
	key := r.Context().Value(middleware.KeyContextKey).(*types.Key)

	idStr := chi.URLParam(r, "id")
	id, err := types.ToSnowflake(idStr)
	if err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	keyToDelete, err := keySvc.GetKeyFromID(id)
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

	if err := keySvc.DeleteKey(id); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
