package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/middleware"
	"reverse-watch/internal/render"
)

func (h *Handler) createReversalsHandler(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	type reversal struct {
		SteamID        models.SteamID  `json:"steam_id"`
		Source         *models.Source  `json:"source"`
		RelatedSteamID *models.SteamID `json:"related_steam_id"`
		ReversedAt     uint64          `json:"reversed_at"`
	}

	var req struct {
		Data []*reversal `json:"data"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	reversals := make([]*models.Reversal, 0)
	for _, reversal := range req.Data {
		if !reversal.SteamID.IsValid() {
			render.Errorf(w, r, errors.BadRequest, "invalid steam id")
			return
		}

		if reversal.RelatedSteamID != nil && !reversal.RelatedSteamID.IsValid() {
			render.Errorf(w, r, errors.BadRequest, "invalid related steam id")
			return
		}

		reversals = append(reversals, &models.Reversal{
			SteamID:         reversal.SteamID,
			MarketplaceSlug: key.MarketplaceSlug,
			Source:          reversal.Source,
			RelatedSteamID:  reversal.RelatedSteamID,
			ReversedAt:      reversal.ReversedAt,
		})
	}

	if err := h.reversalSvc.BulkCreateReversals(reversals); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	render.JSON(w, r, &reversals)
}
