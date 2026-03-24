package users

import (
	"context"
	"net/http"
	"strings"
	"time"

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

func fetchUserStatus(steamWebAPIKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

		steamIdStr := chi.URLParam(r, "steamId")
		client := &http.Client{Timeout: 15 * time.Second}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var steamId *models.SteamID
		var err error
		if strings.TrimSpace(steamWebAPIKey) != "" {
			steamId, err = models.ParseSteamUserInputWithOpts(ctx, client, steamIdStr, &models.SteamUserInputOpts{
				UseWebAPIForVanity: true,
				SteamWebAPIKey:     steamWebAPIKey,
			})
		} else {
			steamId, err = models.ParseSteamUserInput(ctx, client, steamIdStr)
		}
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
}
