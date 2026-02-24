package reversals

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/render"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
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
		OrderParam: &dto.OrderParam{
			Column:    "id",
			Direction: dto.DESC,
		},
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

func exportReversals(w http.ResponseWriter, r *http.Request) {
	factory, ok := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	if !ok {
		render.Errorf(w, r, errors.InternalServerError, "missing factory from context")
		return
	}

	defaultLimit := uint(10_000)
	maxLimit := uint(50_000)

	reversals, nextCursor, err := listReversals(factory, r.URL.Query(), defaultLimit, maxLimit)
	if err != nil {
		render.Error(w, r, err)
		return
	}

	headers := []string{"id", "created_at", "updated_at", "deleted_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at"}
	records := [][]string{headers}

	for _, reversal := range reversals {
		var source string
		if reversal.Source != nil {
			source = reversal.Source.String()
		}

		var relatedSteamID string
		if reversal.RelatedSteamID != nil {
			relatedSteamID = reversal.RelatedSteamID.String()
		}

		var deletedAt string
		if !reversal.DeletedAt.Time.IsZero() {
			deletedAt = strconv.FormatUint(uint64(reversal.DeletedAt.Time.UnixMilli()), 10)
		}

		records = append(records, []string{
			reversal.ID.String(),
			strconv.FormatUint(reversal.CreatedAt, 10),
			strconv.FormatUint(reversal.UpdatedAt, 10),
			deletedAt,
			reversal.SteamID.String(),
			reversal.MarketplaceSlug,
			source,
			relatedSteamID,
			strconv.FormatUint(reversal.ReversedAt, 10),
		})
	}

	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	if err := writer.WriteAll(records); err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to encode csv")
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

func deleteReversal(w http.ResponseWriter, r *http.Request) {
	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)
	key := r.Context().Value(middleware.KeyContextKey).(*models.Key)

	id := chi.URLParam(r, "id")
	if id == "" {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	snowflake, err := models.ToSnowflake(id)
	if err != nil {
		render.Errorf(w, r, errors.BadRequest, "invalid id")
		return
	}

	err = factory.RunInTransactionPublic(func(tx repository.PublicTransaction) error {
		reversal, err := tx.Reversal().Read(snowflake)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return errors.New(errors.NotFound, err.Error())
			}
			return err
		}

		if key.MarketplaceSlug != reversal.MarketplaceSlug {
			return errors.New(errors.BadRequest, "cannot delete reversal report of another marketplace")
		}

		if !reversal.DeletedAt.Time.IsZero() {
			return errors.New(errors.BadRequest, "reversal has already been deleted")
		}

		if err := tx.Reversal().Delete(snowflake); err != nil {
			return err
		}

		logging.Log.Infof("marketplace %s deleted reversal report with id %d", key.MarketplaceSlug, snowflake)
		return nil
	})
	if err != nil {
		if e, ok := err.(*errors.Error); ok {
			render.Error(w, r, e)
			return
		}
		render.Errorf(w, r, errors.InternalServerError, "failed to delete reversal")
		return
	}
	w.WriteHeader(http.StatusOK)
}
