package csfloat

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"reverse-watch/config"
	"reverse-watch/domain/dto"
	"reverse-watch/domain/ingestors"
	"reverse-watch/domain/leader"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/util"

	"go.uber.org/zap"
)

type csfloatIngestor struct {
	log     *zap.SugaredLogger
	cfg     *config.Config
	factory repository.Factory
	elector leader.Elector

	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

var _ ingestors.Ingestor = (*csfloatIngestor)(nil)

func NewCSFloatIngestor(ctx context.Context, factory repository.Factory, elector leader.Elector, cfg *config.Config, log *zap.SugaredLogger) ingestors.Ingestor {
	ingestorCtx, cancel := context.WithCancel(ctx)
	return &csfloatIngestor{
		log:     log,
		cfg:     cfg,
		factory: factory,
		elector: elector,
		ctx:     ingestorCtx,
		cancel:  cancel,
		stopped: make(chan struct{}),
	}
}

type slimWarning struct {
	ID        uint           `json:"id"`
	SteamID   models.SteamID `json:"steam_id"`
	CreatedAt time.Time      `json:"created_at"`
}

type responseData struct {
	Data       []*slimWarning `json:"data"`
	NextCursor *string        `json:"next_cursor"`
}

type errorResponse struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

// fetch reversal warnings from CSFloat
func (i *csfloatIngestor) fetch(ctx context.Context, cursor *uint) ([]*slimWarning, *uint, error) {
	// The day before Valve introduced trade reversals
	startTime := time.Date(2025, 7, 14, 0, 0, 0, 0, time.UTC)
	endTime := time.Now().Add(-5 * time.Minute)
	limit := 1000
	url := fmt.Sprintf("%s/api/v1/warnings/reversals?start_time_ms=%d&end_time_ms=%d&limit=%d", i.cfg.Ingestors.CSFloat.BaseURL, startTime.UnixMilli(), endTime.UnixMilli(), limit)
	if cursor != nil {
		url = fmt.Sprintf("%s&cursor=%s", url, *cursor)
	}

	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	r.Header.Set("X-Secret-Key", i.cfg.Ingestors.CSFloat.SecretKey)
	r = r.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, nil, errors.New(errors.JSONDecode, err.Error())
		}

		i.log.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
		return nil, nil, fmt.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
	}

	var data responseData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, nil, errors.New(errors.JSONDecode, err.Error())
	}

	var nextCursor *uint
	if data.NextCursor != nil {
		next, err := strconv.ParseUint(*data.NextCursor, 10, 64)
		if err != nil {
			return nil, nil, errors.New(errors.JSONDecode, err.Error())
		}
		nextCursor = util.Ptr(uint(next))
	}

	return data.Data, nextCursor, nil
}

// process warnings and create reversals, returns the most recent reversal
func (i *csfloatIngestor) process(warnings []*slimWarning) error {
	for _, warning := range warnings {
		reversal := &models.Reversal{
			SteamID:            warning.SteamID,
			MarketplaceSlug:    "csfloat",
			Source:             util.Ptr(models.SourceDirect),
			ReversedAt:         uint64(warning.CreatedAt.UnixMilli()),
			ReporterInternalID: &warning.ID,
		}

		if err := i.factory.Reversal().Create(reversal); err != nil {
			if errors.IsUniqueConstraintError(err) {
				i.log.Warnf("failed to create reversal %v: %v", reversal, err)
				continue
			}
			return err
		}
	}
	return nil
}

func (i *csfloatIngestor) sync(ctx context.Context) error {
	// Fetch the most recent reversals created by CSFloat
	reversals, err := i.factory.Reversal().List(&dto.ReversalListOptions{
		MarketplaceSlug: util.Ptr("csfloat"),
		Limit:           util.Ptr[uint](1),
		OrderParam: &dto.OrderParam{
			Column:    "reporter_internal_id",
			Direction: dto.DESC,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list recently inserted reversals: %v", err)
	}

	var cursor *uint
	if len(reversals) > 0 {
		cursor = reversals[0].ReporterInternalID
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		warnings, nextCursor, err := i.fetch(ctx, cursor)
		if err != nil {
			return fmt.Errorf("failed to fetch warnings with cursor %v: %v", cursor, err)
		}

		if len(warnings) == 0 {
			return nil
		}

		if err := i.process(warnings); err != nil {
			return fmt.Errorf("failed to process reversals with cursor %v: %v", cursor, err)
		}

		if nextCursor == nil {
			return nil
		}
		cursor = nextCursor
	}
}

func (i *csfloatIngestor) Start() {
	i.log.Info("starting csfloat ingestor")
	go func() {
		defer close(i.stopped)

		h := fnv.New32a()
		h.Write([]byte("csfloat_ingestor_leader"))
		lockKey := h.Sum32()

		i.elector.Run(i.ctx, lockKey, 30*time.Minute, func(ctx context.Context) {
			if err := i.sync(ctx); err != nil {
				i.log.Errorf("failed to sync reversals: %v", err)
			}
		})
	}()
}

func (i *csfloatIngestor) Stop() {
	i.log.Infof("stopping csfloat ingestor")
	i.cancel()
	<-i.stopped
}
