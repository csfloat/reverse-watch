package reversals

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/errors"
	"reverse-watch/internal/middleware"
	"reverse-watch/internal/render"

	"github.com/go-chi/chi/v5"
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
		render.Errorf(w, r, errors.DBCreate, "failed to create bulk reversals")
		return
	}

	render.JSON(w, r, &reversals)
}

func (h *Handler) expungeReversalHandler(w http.ResponseWriter, r *http.Request) {
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	id := chi.URLParam(r, "id")
	if id == "" {
		render.Error(w, r, &errors.BadRequest)
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	reversal, err := h.reversalSvc.GetReversal(snowflake)
	if err != nil {
		render.Errorf(w, r, errors.UnknownResource, "reversal not found")
		return
	}

	if key.MarketplaceSlug != reversal.MarketplaceSlug {
		render.Errorf(w, r, errors.NotAuthorized, "marketplace mismatch")
	}

	if err := h.reversalSvc.ExpungeReversal(snowflake); err != nil {
		render.Errorf(w, r, errors.DBCreate, "failed to expunge reversal")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) listReversals(queryValues url.Values, defaultLimit, maxLimit uint) ([]*models.Reversal, *models.Cursor, error) {
	listOpts := &repository.ReversalListOptions{
		Limit: &defaultLimit,
	}

	if steamIDStr := queryValues.Get("steam_id"); steamIDStr != "" {
		steamID, err := models.ToSteamID(steamIDStr)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid steam id")
		}
		listOpts.SteamID = steamID
	}

	if marketplaceSlugStr := queryValues.Get("marketplace_slug"); marketplaceSlugStr != "" {
		listOpts.MarketplaceSlug = &marketplaceSlugStr
	}

	if limitStr := queryValues.Get("limit"); limitStr != "" {
		limit64, err := strconv.ParseUint(limitStr, 10, 64)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid limit")
		}
		limit := uint(limit64)
		if limit > maxLimit {
			return nil, nil, errors.Newf(errors.BadRequest, nil, "limit exceeds max limit of %d", maxLimit)
		}
		listOpts.Limit = &limit
	}

	if cursorStr := queryValues.Get("cursor"); cursorStr != "" {
		cursor, err := models.DecodeCursor(cursorStr)
		if err != nil {
			return nil, nil, errors.New(errors.BadRequest, "invalid cursor")
		}
		listOpts.Cursor = cursor
	}

	reversals, err := h.reversalSvc.ListReversals(listOpts)
	if err != nil {
		return nil, nil, errors.New(errors.InternalServerError, "failed to list reversals")
	}

	var nextCursor *models.Cursor
	if len(reversals) != 0 {
		// Omit cursor on last page
		if len(reversals) == int(*listOpts.Limit) {
			nextCursor = &models.Cursor{
				ID:         reversals[len(reversals)-1].ID,
				ReversedAt: reversals[len(reversals)-1].ReversedAt,
			}
		}
	}

	return reversals, nextCursor, nil
}

func (h *Handler) listReversalsHandler(w http.ResponseWriter, r *http.Request) {
	defaultLimit := uint(5_000)
	maxLimit := uint(10_000)

	reversals, nextCursor, err := h.listReversals(r.URL.Query(), defaultLimit, maxLimit)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	type metadata struct {
		Count      uint           `json:"count"`
		NextCursor *models.Cursor `json:"next_cursor,omitempty"`
	}

	type resp struct {
		Data     []*models.Reversal `json:"data"`
		Metadata metadata           `json:"metadata"`
	}

	render.JSON(w, r, &resp{
		Data: reversals,
		Metadata: metadata{
			Count:      uint(len(reversals)),
			NextCursor: nextCursor,
		},
	})
}

func (h *Handler) exportReversalsHandler(w http.ResponseWriter, r *http.Request) {
	defaultLimit := uint(10_000)
	maxLimit := uint(50_000)

	reversals, nextCursor, err := h.listReversals(r.URL.Query(), defaultLimit, maxLimit)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	headers := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
	data := [][]string{headers}

	for _, reversal := range reversals {
		var source string
		if reversal.Source != nil {
			source = reversal.Source.String()
		}

		var relatedSteamID string
		if reversal.RelatedSteamID != nil {
			relatedSteamID = reversal.RelatedSteamID.String()
		}

		var expungedAt string
		if reversal.ExpungedAt != nil {
			expungedAt = strconv.FormatUint(*reversal.ExpungedAt, 10)
		}

		data = append(data, []string{
			reversal.ID.String(),
			strconv.FormatUint(reversal.CreatedAt, 10),
			strconv.FormatUint(reversal.UpdatedAt, 10),
			reversal.SteamID.String(),
			reversal.MarketplaceSlug,
			source,
			relatedSteamID,
			strconv.FormatUint(reversal.ReversedAt, 10),
			expungedAt,
		})
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.WriteAll(data); err != nil {
		render.Error(w, r, &errors.CSVEncode)
		return
	}

	if nextCursor != nil {
		encodedCursor, err := nextCursor.Encode()
		if err != nil {
			render.Errorf(w, r, errors.InternalServerError, "failed to encode cursor")
			return
		}
		w.Header().Set("X-Next-Cursor", *encodedCursor)
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Write(buf.Bytes())
}
