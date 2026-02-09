package reversals

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
)

func createReversals(w http.ResponseWriter, r *http.Request) {
	factory, ok := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing factory from context")
		return
	}

	key, ok := r.Context().Value(middleware.KeyContextKey).(*models.Key)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing key from context")
		return
	}

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
		render.Error(w, r, &errors.JSONDecode)
		return
	}

	reversals := make([]*models.Reversal, 0)
	for _, reversal := range req.Data {
		reversals = append(reversals, &models.Reversal{
			SteamID:         reversal.SteamID,
			MarketplaceSlug: key.MarketplaceSlug,
			Source:          reversal.Source,
			RelatedSteamID:  reversal.RelatedSteamID,
			ReversedAt:      reversal.ReversedAt,
		})
	}

	if err := factory.Reversal().BulkCreate(reversals); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to create reversals")
		return
	}

	render.JSON(w, r, &reversals)
}

func listReversals(f repository.Factory, values url.Values, defaultLimit, maxLimit uint) ([]*models.Reversal, *dto.Cursor, error) {
	opts := &dto.ReversalListOptions{
		Limit: &defaultLimit,
	}

	if steamIdStr := values.Get("steam_id"); steamIdStr != "" {
		steamId, err := models.ToSteamID(steamIdStr)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid steam id")
		}
		opts.SteamID = steamId
	}

	if marketplaceStr := values.Get("marketplace_slug"); marketplaceStr != "" {
		opts.MarketplaceSlug = &marketplaceStr
	}

	if limitStr := values.Get("limit"); limitStr != "" {
		limit64, err := strconv.ParseUint(limitStr, 10, 64)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid limit")
		}
		limit := uint(limit64)
		if limit > maxLimit {
			return nil, nil, errors.Newf(errors.BadRequest, nil, "limit exceeds max limit of %d", maxLimit)
		}
		opts.Limit = &limit
	}

	if cursorStr := values.Get("cursor"); cursorStr != "" {
		cursor, err := dto.DecodeCursor(cursorStr)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid cursor")
		}
		opts.Cursor = cursor
	}

	reversals, err := f.Reversal().List(opts)
	if err != nil {
		return nil, nil, errors.New(errors.InternalServerError, "failed to list reversals")
	}

	var nextCursor *dto.Cursor
	if len(reversals) > 0 {
		// Omit cursor on last page
		if len(reversals) == int(*opts.Limit) {
			nextCursor = &dto.Cursor{
				ID: reversals[len(reversals)-1].ID,
			}
		}
	}
	return reversals, nextCursor, nil
}

type metadata struct {
	Count      int         `json:"count"`
	NextCursor *dto.Cursor `json:"next_cursor,omitempty"`
}

type listReversalsResponse struct {
	Data     []*models.Reversal `json:"data"`
	Metadata metadata           `json:"metadata"`
}

func listReversalsHandler(w http.ResponseWriter, r *http.Request) {
	factory, ok := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing factory from context")
		return
	}

	defaultLimit := uint(5_000)
	maxLimit := uint(10_000)

	reversals, nextCursor, err := listReversals(factory, r.URL.Query(), defaultLimit, maxLimit)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	render.JSON(w, r, &listReversalsResponse{
		Data: reversals,
		Metadata: metadata{
			Count:      len(reversals),
			NextCursor: nextCursor,
		},
	})
}
