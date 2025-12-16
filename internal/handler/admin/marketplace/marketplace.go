package marketplace

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) createMarketplace(w http.ResponseWriter, r *http.Request) {
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

	marketplace := &models.Marketplace{
		Slug:     req.MarketplaceSlug,
		Name:     req.Name,
		IsActive: true,
	}

	if err := h.marketplaceSvc.CreateMarketplace(marketplace); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	rawKey, err := h.keySvc.CreateKey(marketplace.Slug, models.PermissionManage)
	if err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	storedMarketplace, err := h.marketplaceSvc.GetMarketplace(marketplace.Slug)
	if err != nil {
		render.Error(w, r, &errors.DBRead)
		return
	}

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction: models.TargetActionAddMarketplace,
		Details: &models.Jsonb{
			"slug": storedMarketplace.Slug,
		},
	}); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	render.JSON(w, r, struct {
		Marketplace *models.Marketplace `json:"marketplace"`
		Key         *models.RawKey      `json:"key"`
	}{
		Marketplace: storedMarketplace,
		Key:         rawKey,
	})
}

func (h *Handler) patchMarketplace(w http.ResponseWriter, r *http.Request) {
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

	if err := h.marketplaceSvc.UpdateMarketplace(slug, fields); err != nil {
		render.Errorf(w, r, errors.DBUpdate, "failed to update marketplace")
		return
	}

	marketplace, err := h.marketplaceSvc.GetMarketplace(slug)
	if err != nil {
		render.Errorf(w, r, errors.DBRead, "failed to find marketplace")
		return
	}

	// Construct details for admin audit
	details := models.Jsonb{
		"slug": slug,
	}
	for k, v := range fields {
		details[k] = v
	}

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction: models.TargetActionUpdateMarketplace,
		Details:      &details,
	}); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to create admin audit")
		return
	}

	render.JSON(w, r, marketplace)
}

func (h *Handler) deleteMarketplace(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if slug == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if err := h.marketplaceSvc.DeleteMarketplace(slug); err != nil {
		render.Errorf(w, r, errors.DBDelete, "failed to delete marketplace")
		return
	}

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction: models.TargetActionRemoveMarketplace,
		Details: &models.Jsonb{
			"slug": slug,
		},
	}); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to create admin audit")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
