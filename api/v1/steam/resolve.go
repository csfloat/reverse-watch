package steam

import (
	"net/http"
	"strings"

	"reverse-watch/domain/models"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
	steamservice "reverse-watch/service/steam"
)

type resolveSteamIDResponse struct {
	SteamID models.SteamID `json:"steam_id"`
}

func resolveSteamID(w http.ResponseWriter, r *http.Request) {
	steamSvc, ok := r.Context().Value(middleware.SteamServiceContextKey).(*steamservice.Service)
	if !ok || steamSvc == nil {
		render.Errorf(w, r, errors.InternalServerError, "steam service not configured")
		return
	}

	raw := strings.TrimSpace(r.URL.Query().Get("vanityUrl"))
	if raw == "" {
		render.Errorf(w, r, errors.BadRequest, "missing vanityUrl")
		return
	}

	id, err := steamSvc.ResolveSteamID(r.Context(), raw)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	render.JSON(w, r, &resolveSteamIDResponse{SteamID: *id})
}
