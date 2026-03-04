package csfloat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reverse-watch/config"
	"reverse-watch/domain/dto"
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

	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

func NewCSFloatIngestor(ctx context.Context, factory repository.Factory, cfg *config.Config, logger *zap.SugaredLogger) *csfloatIngestor {
	ingestorCtx, cancel := context.WithCancel(ctx)
	return &csfloatIngestor{
		log:     logger,
		cfg:     cfg,
		factory: factory,
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
	NextCursor *uint          `json:"next_cursor"`
}

type errorResponse struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

// fetch reversal warnings from CSFloat
func (i *csfloatIngestor) fetch(cursor *uint, startTime, endTime time.Time, limit uint) ([]*slimWarning, *uint, error) {
	url := fmt.Sprintf("%s/api/v1/warnings/reversals?&start_time_ms=%d&end_time_ms=%d&limit=%d", i.cfg.Ingestors.CSFloat.BaseURL, startTime.UnixMilli(), endTime.UnixMilli(), limit)
	if cursor != nil {
		url = fmt.Sprintf("%s&cursor=%d", url, *cursor)
	}

	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}
	r.Header.Set("X-Secret-Key", i.cfg.Ingestors.CSFloat.SecretKey)
	r = r.WithContext(i.ctx)

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

	return data.Data, data.NextCursor, nil
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

func (i *csfloatIngestor) sync() error {
	// Fetch the most recent reversals created by CSFloat
	reversals, err := i.factory.Reversal().List(&dto.ReversalListOptions{
		MarketplaceSlug: util.Ptr("csfloat"),
		Limit:           util.Ptr[uint](1),
		OrderParam: &dto.OrderParam{
			Column:    "id",
			Direction: dto.DESC,
		},
	})
	if err != nil {
		i.log.Errorf("failed to list recently inserted reversals: %v", err)
		return fmt.Errorf("failed to list recently inserted reversals: %v", err)
	}

	var cursor *uint
	if len(reversals) > 0 {
		cursor = reversals[0].ReporterInternalID
	}

	for {
		select {
		case <-i.ctx.Done():
			return nil
		default:
		}

		// The day before Valve introduced trade reversals
		startTime := time.Date(2025, 7, 14, 0, 0, 0, 0, time.UTC)
		limit := 2
		warnings, nextCursor, err := i.fetch(cursor, startTime, time.Now().Add(-5*time.Minute), uint(limit))
		if err != nil {
			i.log.Errorf("failed to fetch warnings with cursor %v: %v", cursor, err)
			return fmt.Errorf("failed to fetch warnings with cursor %v: %v", cursor, err)
		}

		if len(warnings) == 0 {
			select {
			case <-time.After(1 * time.Minute):
				continue
			case <-i.ctx.Done():
				return nil
			}
		}

		if err := i.process(warnings); err != nil {
			i.log.Errorf("failed to process reversals: %v", err)
			return fmt.Errorf("failed to process reversals: %v", err)
		}

		if nextCursor == nil && len(warnings) < limit {
			select {
			case <-time.After(30 * time.Minute):
				continue
			case <-i.ctx.Done():
				return nil
			}
		}
		cursor = nextCursor
	}
}

func (i *csfloatIngestor) Start() {
	i.log.Info("Starting CSFloat ingestor")
	go func() {
		defer close(i.stopped)

		sleepTime := time.Minute
		for {
			if err := i.sync(); err != nil {
				i.log.Errorf("failed to sync reversals: %v", err)
				select {
				case <-time.After(sleepTime):
					sleepTime = min(sleepTime*2, 30*time.Minute)
				case <-i.ctx.Done():
					return
				}
			} else {
				select {
				case <-i.ctx.Done():
					return
				default:
					sleepTime = time.Minute
				}
			}
		}
	}()
}

func (i *csfloatIngestor) Stop() {
	i.log.Infof("Stopping CSFloat ingestor")
	i.cancel()
	<-i.stopped
}

func (i *csfloatIngestor) Done() <-chan struct{} {
	return i.stopped
}
