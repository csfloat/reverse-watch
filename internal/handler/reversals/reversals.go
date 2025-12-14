package reversals

import (
	"encoding/json"
	"net/http"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/middleware"
	"reverse-watch/internal/render"
)

func (h *Handler) createReversalHandler(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	var req struct {
		SteamID    models.SteamID `json:"steam_id"`
		ReversedAt uint64         `json:"reversed_at"`
	}

	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	if !req.SteamID.IsValid() {
		render.Errorf(w, r, errors.BadRequest, "invalid steam id")
		return
	}

	reversal := &models.Reversal{
		SteamID:         req.SteamID,
		MarketplaceSlug: key.MarketplaceSlug,
		ReversedAt:      req.ReversedAt,
	}

	if err := h.reversalSvc.CreateReversal(reversal); err != nil {
		render.Error(w, r, &errors.DBCreate)
		return
	}

	render.JSON(w, r, reversal)
}
