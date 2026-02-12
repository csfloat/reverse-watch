package users

import (
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
)

func fetchUserStatus(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	steamIdStr := chi.URLParam(r, "steamId")
	steamId, err := models.ToSteamID(steamIdStr)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "invalid steam id")
		return
	}

	reversals, err := factory.Reversal().List(&dto.ReversalListOptions{
		SteamID: steamId,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list reversals for steam id %q", steamId)
		return
	}

	resp := struct {
		SteamID               models.SteamID `json:"steam_id"`
		HasReversed           bool           `json:"has_reversed"`
		IsExpunged            bool           `json:"is_expunged"`
		LastReversalTimestamp *uint64        `json:"last_reversal_timestamp,omitempty"`
	}{
		SteamID: *steamId,
	}

	if len(reversals) > 0 {
		resp.HasReversed = true
		// Reversals are sorted in descending order by ID
		if reversals[0].ExpungedAt != nil && *reversals[0].ExpungedAt > 0 {
			resp.IsExpunged = true
		}
		resp.LastReversalTimestamp = &reversals[0].ReversedAt
	}
	render.JSON(w, r, resp)
}
