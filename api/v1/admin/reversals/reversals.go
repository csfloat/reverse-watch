package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

func modifyReversal(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	authKey := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	id := chi.URLParam(r, "id")
	if id == "" {
		render.Errorf(w, r, errors.BadRequest, "id must not be empty")
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	var opts dto.ReversalUpdates
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	if err := factory.Reversal().Update(snowflake, &opts); err != nil {
		render.Error(w, r, err)
		return
	}

	details, err := models.ToRawJsonb(opts)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	audit := models.NewReversalAdminAudit(models.TargetActionUpdateReversal, snowflake, authKey.ID, details)
	if err := factory.AdminAudit().Create(audit); err != nil {
		render.Error(w, r, err)
		return
	}

	reversal, err := factory.Reversal().Read(snowflake)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	render.JSON(w, r, reversal)
}
