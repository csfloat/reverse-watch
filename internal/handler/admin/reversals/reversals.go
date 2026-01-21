package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
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

	var opts dto.ReversalUpdateOptions

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if err := h.reversalSvc.UpdateReversal(snowflake, &opts); err != nil {
		render.Errorf(w, r, errors.DBUpdate, "failed to update reversal with id %q: %v", snowflake.String(), err)
		return
	}

	details, err := models.ToRawJsonb(opts)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to convert to raw jsonb")
		return
	}

	audit := models.NewReversalAdminAudit(models.TargetActionUpdateMarketplace, snowflake, details)
	if err := h.adminAuditSvc.CreateAdminAudit(audit); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to create admin audit")
		return
	}

	reversal, err := h.reversalSvc.GetReversal(snowflake)
	if err != nil {
		render.Errorf(w, r, errors.DBRead, "failed to get updated reversal with id %q", snowflake.String())
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
		render.Errorf(w, r, errors.DBDelete, "failed to delete reversal with id %q", snowflake.String())
		return
	}

	if err := h.adminAuditSvc.CreateAdminAudit(&models.AdminAudit{
		TargetAction:       models.TargetActionRemoveReversal,
		TargetResourceType: models.TargetResourceTypeReversal,
		TargetResource:     snowflake.String(),
	}); err != nil {
		render.Errorf(w, r, errors.DBDelete, "failed to create admin audit")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
