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

	cachedSteamIDs map[models.SteamID]struct{}

	ctx     context.Context
	cancel  context.CancelFunc
	stopped chan struct{}
}

func NewCSFloatIngestor(ctx context.Context, factory repository.Factory, cfg *config.Config, logger *zap.SugaredLogger) *csfloatIngestor {
	ingestorCtx, cancel := context.WithCancel(ctx)
	return &csfloatIngestor{
		log:            logger,
		cfg:            cfg,
		factory:        factory,
		cachedSteamIDs: make(map[models.SteamID]struct{}),
		ctx:            ingestorCtx,
		cancel:         cancel,
		stopped:        make(chan struct{}),
	}
}

type slimWarning struct {
	ID        uint           `json:"id"`
	SteamID   models.SteamID `json:"steam_id"`
	CreatedAt time.Time      `json:"created_at"`
}

type responseData struct {
	Data       []*slimWarning `json:"data"`
	NextCursor uint           `json:"next_cursor"`
}

type errorResponse struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

// fetch reversal warnings from CSFloat
func (i *csfloatIngestor) fetch(cursor uint, startTime, endTime time.Time) ([]*slimWarning, uint, error) {
	url := fmt.Sprintf("%s/api/v1/warnings/reversals?cursor=%d&start_time_ms=%d&end_time_ms=%d&limit=%d", i.cfg.Ingestors.CSFloat.BaseURL, cursor, startTime.UnixMilli(), endTime.UnixMilli(), 1000)
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, err
	}
	r.Header.Set("X-Secret-Key", i.cfg.Ingestors.CSFloat.SecretKey)
	r = r.WithContext(i.ctx)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, 0, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, 0, errors.New(errors.JSONDecode, err.Error())
		}

		i.log.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
		return nil, 0, fmt.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
	}

	var data responseData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, 0, errors.New(errors.JSONDecode, err.Error())
	}

	return data.Data, data.NextCursor, nil
}

// process warnings and create reversals, returns the most recent reversed at time if any warnings were processed
func (i *csfloatIngestor) process(warnings []*slimWarning) time.Time {
	var mostRecent time.Time
	for _, warning := range warnings {
		if _, ok := i.cachedSteamIDs[warning.SteamID]; ok {
			continue
		}
		i.cachedSteamIDs[warning.SteamID] = struct{}{}

		reversal := &models.Reversal{
			SteamID:            warning.SteamID,
			MarketplaceSlug:    "csfloat",
			Source:             util.Ptr(models.SourceDirect),
			ReversedAt:         uint64(warning.CreatedAt.UnixMilli()),
			ReporterInternalID: &warning.ID,
		}

		if err := i.factory.Reversal().Create(reversal); err != nil {
			i.log.Errorf("failed to create reversal %v: %v", reversal, err)
			continue
		}

		if warning.CreatedAt.After(mostRecent) {
			mostRecent = warning.CreatedAt
		}
	}
	return mostRecent
}

func (i *csfloatIngestor) sync() error {
	// Fetch the most recent reversals created by CSFloat
	reversals, err := i.factory.Reversal().List(&dto.ReversalListOptions{
		MarketplaceSlug: util.Ptr("csfloat"),
		Limit:           util.Ptr[uint](50),
		OrderParam: &dto.OrderParam{
			Column:    "id",
			Direction: dto.DESC,
		},
	})
	if err != nil {
		i.log.Errorf("failed to list recently inserted reversals: %v", err)
		return fmt.Errorf("failed to list recently inserted reversals: %v", err)
	}

	var cursor uint
	for _, reversal := range reversals {
		if reversal.ReporterInternalID != nil {
			cursor = max(cursor, *reversal.ReporterInternalID)
		}

		if _, ok := i.cachedSteamIDs[reversal.SteamID]; ok {
			continue
		}
		i.cachedSteamIDs[reversal.SteamID] = struct{}{}
	}

	for {
		select {
		case <-i.ctx.Done():
			return nil
		default:
		}

		warnings, nextCursor, err := i.fetch(cursor, time.Time{}, time.Now().Add(-5*time.Minute))
		if err != nil {
			i.log.Errorf("failed to fetch warnings with cursor %v: %v", cursor, err)
			return fmt.Errorf("failed to fetch warnings with cursor %v: %v", cursor, err)
		}

		mostRecent := i.process(warnings)
		if mostRecent.After(time.Now().Add(-30 * time.Minute)) {
			select {
			case <-time.After(30 * time.Minute):
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

func (i *csfloatIngestor) IsEnabled() bool {
	return i.cfg.Ingestors.CSFloat.Enable
}
