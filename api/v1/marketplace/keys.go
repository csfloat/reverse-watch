package marketplace

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
)

func createKey(w http.ResponseWriter, r *http.Request) {
	factory, ok := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing factory from context")
		return
	}

	key, ok := r.Context().Value(middleware.KeyContextKey).(*models.Key)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing key from context")
		return
	}

	var req struct {
		Permissions models.Permissions `json:"permissions"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.Permissions == 0 {
		render.Errorf(w, r, errors.BadRequest, "invalid permissions")
		return
	}

	if req.Permissions.HasPermissions(models.PermissionAdmin) && !key.IsOwnedByCSFloat() {
		render.Errorf(w, r, errors.BadRequest, "admin scoped keys can only be created for csfloat")
		return
	}

	rawKey, err := factory.Key().Create(key.MarketplaceSlug, req.Permissions)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to create key")
		return
	}

	render.JSON(w, r, rawKey)
}

func listKeys(w http.ResponseWriter, r *http.Request) {
	factory, ok := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing factory from context")
		return
	}

	key, ok := r.Context().Value(middleware.KeyContextKey).(*models.Key)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing key from context")
		return
	}

	keysList, err := factory.Key().List(&dto.KeyListOptions{
		MarketplaceSlug: &key.MarketplaceSlug,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list keys")
		return
	}

	render.JSON(w, r, keysList)
}
