package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) patchReversalHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	var req struct {
		SteamID         *models.SteamID `json:"steam_id"`
		MarketplaceSlug *string         `json:"marketplace_slug"`
		Source          *models.Source  `json:"source"`
		RelatedSteamID  *models.SteamID `json:"related_steam_id"`
		ReversedAt      *uint64         `json:"reversed_at"`
		ExpungedAt      *uint64         `json:"expunged_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	opts := &repository.ReversalUpdateOptions{}
	if req.SteamID != nil {
		opts.SteamID = req.SteamID
	}
	if req.MarketplaceSlug != nil {
		opts.MarketplaceSlug = req.MarketplaceSlug
	}
	if req.Source != nil {
		opts.Source = req.Source
	}
	if req.RelatedSteamID != nil {
		opts.RelatedSteamID = req.RelatedSteamID
	}
	if req.ReversedAt != nil {
		opts.ReversedAt = req.ReversedAt
	}
	if req.ExpungedAt != nil {
		opts.ExpungedAt = req.ExpungedAt
	}

	if err := h.reversalSvc.UpdateReversal(snowflake, opts); err != nil {
		render.Errorf(w, r, errors.DBUpdate, "failed to update reversal with id %q", snowflake)
		return
	}

	details := models.Jsonb(opts.ToFields())

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction:   models.TargetActionUpdateReversal,
		TargetResource: &snowflake,
		Details:        &details,
	}); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to create admin audit")
		return
	}

	reversal, err := h.reversalSvc.GetReversal(snowflake)
	if err != nil {
		render.Errorf(w, r, errors.DBRead, "failed to get reversal with id %q", snowflake)
		return
	}

	render.JSON(w, r, reversal)
}

func (h *Handler) deleteReversalHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	if err := h.reversalSvc.DeleteReversal(snowflake); err != nil {
		render.Errorf(w, r, errors.DBDelete, "failed to delete reversal with id %q", snowflake)
		return
	}

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction:   models.TargetActionRemoveReversal,
		TargetResource: &snowflake,
	}); err != nil {
		render.Errorf(w, r, errors.DBDelete, "failed to create admin audit")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
