package marketplace

import (
	"encoding/json"
	"net/http"

	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
	"reverse-watch/services/private"
	"reverse-watch/types"
)

func createMarketplace(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)

	var req struct {
		MarketplaceSlug string `json:"marketplace_slug"`
		Name            string `json:"name"`
	}

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	marketplace := &types.Marketplace{
		Slug:     req.MarketplaceSlug,
		Name:     req.Name,
		IsActive: true,
	}

	if err := privateSvc.CreateMarketplace(marketplace); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	rawKey, err := private.NewRawKey(marketplace.Slug, types.ScopeManage)
	if err != nil {
		render.Error(w, r, &errors.InternalServerError)
		return
	}

	if err := privateSvc.CreateKey(rawKey.ToKey()); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	type key struct {
		*private.RawKey
		Scope string `json:"scope"`
	}

	render.JSON(w, r, struct {
		Marketplace types.Marketplace `json:"marketplace"`
		Key         key               `json:"key"`
	}{
		Marketplace: *marketplace,
		Key: key{
			RawKey: rawKey,
			Scope:  rawKey.Scope.String(),
		},
	})
}
