package users

import (
	"net/http"

	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

func purgeUser(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	authKey := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	steamIdStr := chi.URLParam(r, "steamId")
	steamId, err := models.ToSteamID(steamIdStr)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "%v", err)
		return
	}

	if err := factory.Reversal().DeleteAllUserReports(*steamId); err != nil {
		logging.Log.Errorf("failed to delete all user reports: %v", err)
		render.Error(w, r, err)
		return
	}

	audit := models.NewUserAdminAudit(models.TargetActionDeleteUserData, *steamId, authKey.ID, nil)
	if err := factory.AdminAudit().Create(audit); err != nil {
		logging.Log.Errorf("failed to create admin audit: %v", err)
		render.Error(w, r, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
