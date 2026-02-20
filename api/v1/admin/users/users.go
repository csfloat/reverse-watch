package users

import (
	"net/http"

	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

func purgeUser(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	steamIdStr := chi.URLParam(r, "steamId")
	steamId, err := models.ToSteamID(steamIdStr)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "%v", err)
		return
	}

	if err := factory.Reversal().DeleteUser(*steamId); err != nil {
		render.Error(w, r, err)
		return
	}

	details := struct {
		Key string `json:"key"`
	}{
		Key: key.ID,
	}

	jsonb, err := models.ToRawJsonb(details)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	audit := models.NewUserAdminAudit(models.TargetActionDeleteUserData, *steamId, jsonb)
	if err := factory.AdminAudit().Create(audit); err != nil {
		render.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
