package users

import (
	"net/http"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm/clause"
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
		SteamID: steamId,
		OrderBy: &clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "id"}, Desc: true}}},
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list reversals for steam id %q", steamId)
		return
	}

	// Record the lookup for the "Steam IDs Searched" KPI. This is analytics
	// only, so a failure here must never fail the user-facing lookup: log and
	// continue.
	if err := factory.SearchCount().Increment(*steamId); err != nil {
		logging.Log.Errorf("failed to increment search count for steam id %q: %v", steamId, err)
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
