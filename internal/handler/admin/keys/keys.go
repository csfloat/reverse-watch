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
		render.Errorf(w, r, errors.BadRequest, "marketplace slug and permissions are required")
		return
	}

	rawKey, err := h.keySvc.CreateKey(req.MarketplaceSlug, req.Permissions)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to create key")
		return
	}

	jsonb, err := models.ToRawJsonb(req)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to convert to raw jsonb")
		return
	}

	audit := models.NewKeyAdminAudit(models.TargetActionAddKey, rawKey.ID, jsonb)
	if err := h.adminAuditSvc.CreateAdminAudit(audit); err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to create admin audit")
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

	// Fetch key and add details to admin audit
	key, err := h.keySvc.GetKey(id)
	if err != nil {
		render.Error(w, r, &errors.DBRead)
		return
	}

	if err := h.keySvc.DeleteKey(id); err != nil {
		render.Error(w, r, &errors.DBDelete)
		return
	}

	details := map[string]interface{}{
		"marketplace_slug": key.MarketplaceSlug,
		"permissions":      key.Permissions,
	}

	jsonb, err := models.ToRawJsonb(details)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to convert to raw jsonb")
		return
	}

	audit := models.NewKeyAdminAudit(models.TargetActionRemoveKey, id, jsonb)
	if err := h.adminAuditSvc.CreateAdminAudit(audit); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
