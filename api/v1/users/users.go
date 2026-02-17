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

type fetchUserStatusResponse struct {
	SteamID               models.SteamID `json:"steam_id"`
	HasReversed           bool           `json:"has_reversed"`
	LastReversalTimestamp *uint64        `json:"last_reversal_timestamp,omitempty"`
}

func fetchUserStatus(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	steamIdStr := chi.URLParam(r, "steamId")
	steamId, err := models.ToSteamID(steamIdStr)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid steam id")
		return
	}

	reversals, err := factory.Reversal().List(&dto.ReversalListOptions{
		SteamID:    steamId,
		OrderParam: &dto.OrderParam{Column: "id", Direction: dto.DESC},
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list reversals for steam id %q", steamId)
		return
	}

	data := &fetchUserStatusResponse{
		SteamID: *steamId,
	}

	var lastReversalTime uint64
	for _, reversal := range reversals {
		lastReversalTime = max(lastReversalTime, reversal.ReversedAt)
		if reversal.ExpungedAt == nil {
			data.HasReversed = true
			break
		}
	}
	if data.HasReversed {
		data.LastReversalTimestamp = &lastReversalTime
	}
	render.JSON(w, r, data)
}
