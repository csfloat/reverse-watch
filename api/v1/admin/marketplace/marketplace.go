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

func createMarketplace(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)

	var req struct {
		MarketplaceSlug string `json:"marketplace_slug"`
		Name            string `json:"name"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.MarketplaceSlug == "" || req.Name == "" {
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

	storedMarketplace, err := privateSvc.GetMarketplace(marketplace.Slug)
	if err != nil {
		render.Error(w, r, &errors.DBRead)
		return
	}

	render.JSON(w, r, struct {
		Marketplace *types.Marketplace `json:"marketplace"`
		Key         *private.RawKey    `json:"key"`
	}{
		Marketplace: storedMarketplace,
		Key:         rawKey,
	})
}

func patchMarketplace(w http.ResponseWriter, r *http.Request) {
	privateSvc := r.Context().Value(middleware.PrivateServiceContextKey).(*private.Service)

	slug := chi.URLParam(r, "slug")
	if slug == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	var req struct {
		Name     *string `json:"name"`
		IsActive *bool   `json:"is_active"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if req.IsActive == nil && req.Name == nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	fields := make(map[string]interface{})
	if req.Name != nil && *req.Name != "" {
		fields["name"] = *req.Name
	}
	if req.IsActive != nil {
		fields["is_active"] = *req.IsActive
	}

	if err := privateSvc.UpdateMarketplace(slug, fields); err != nil {
		render.Error(w, r, &errors.DBUpdate)
		return
	}

	marketplace, err := privateSvc.GetMarketplace(slug)
	if err != nil {
		render.Error(w, r, &errors.DBRead)
		return
	}

	render.JSON(w, r, marketplace)
}
