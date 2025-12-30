package users

import (
	"net/http"
	"sort"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getReversalStatus(w http.ResponseWriter, r *http.Request) {
	steamIDStr := chi.URLParam(r, "steam_id")
	if steamIDStr == "" {
		render.Errorf(w, r, errors.BadRequest, "steam id is required")
		return
	}

	steamID, err := models.ToSteamID(steamIDStr)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid steam id")
		return
	}

	reversals, err := h.reversalSvc.ListReversals(&dto.ReversalListOptions{
		SteamID: steamID,
	})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to list reversals for steam id %q", steamIDStr)
		return
	}

	resp := struct {
		SteamID               models.SteamID `json:"steam_id"`
		HasReversed           bool           `json:"has_reversed"`
		IsExpunged            bool           `json:"is_expunged"`
		LastReversalTimestamp *uint64        `json:"last_reversal_timestamp,omitempty"`
	}{
		SteamID: *steamID,
	}

	if len(reversals) > 0 {
		sort.Slice(reversals, func(i, j int) bool {
			return reversals[i].ReversedAt > reversals[j].ReversedAt
		})

		if reversals[0].ExpungedAt != nil && *reversals[0].ExpungedAt > 0 {
			resp.IsExpunged = true
		}
		resp.HasReversed = true
		resp.LastReversalTimestamp = &reversals[0].ReversedAt
	}

	render.JSON(w, r, resp)
}
