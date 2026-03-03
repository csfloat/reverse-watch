package ingestors

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

func New(factory repository.Factory, cfg config.Config, logger *zap.SugaredLogger) *csfloatIngestor {
	ctx, cancel := context.WithCancel(context.Background())
	return &csfloatIngestor{
		log:            logger,
		cfg:            &cfg,
		factory:        factory,
		cachedSteamIDs: make(map[models.SteamID]struct{}),
		ctx:            ctx,
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
	Data []*slimWarning `json:"data"`
}

type errorResponse struct {
	Code    uint64 `json:"code"`
	Message string `json:"message"`
}

// fetch reversal warnings from CSFloat
func (i *csfloatIngestor) fetch(startTime, endTime time.Time) ([]*slimWarning, error) {
	url := fmt.Sprintf("%s/api/v1/warnings/reversals?start_time_ms=%d&end_time_ms=%d&limit=", i.cfg.CSFloat.BaseURL, startTime.UnixMilli(), endTime.UnixMilli())
	r, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	r.Header.Set("X-Secret-Key", i.cfg.CSFloat.SecretKey)
	r = r.WithContext(i.ctx)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errResp errorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, errors.New(errors.JSONDecode, err.Error())
		}

		i.log.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
		return nil, fmt.Errorf("fetching failed with status code %d: %s", errResp.Code, errResp.Message)
	}

	var data responseData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, errors.New(errors.JSONDecode, err.Error())
	}

	return data.Data, nil
}

// process warnings and create reversals, returns the most recent reversed at time if any warnings were processed
func (i *csfloatIngestor) process(warnings []*slimWarning) time.Time {
	var mostRecentReversedAt time.Time
	for _, warning := range warnings {
		if _, ok := i.cachedSteamIDs[warning.SteamID]; ok {
			continue
		}
		i.cachedSteamIDs[warning.SteamID] = struct{}{}

		reversedAt := warning.CreatedAt
		reversal := &models.Reversal{
			SteamID:         warning.SteamID,
			MarketplaceSlug: "csfloat",
			Source:          util.Ptr(models.SourceDirect),
			ReversedAt:      uint64(reversedAt.UnixMilli()),
		}

		if err := i.factory.Reversal().Create(reversal); err != nil {
			i.log.Errorf("failed to create reversal %v: %v", reversal, err)
			continue
		}

		if reversedAt.After(mostRecentReversedAt) {
			mostRecentReversedAt = reversedAt
		}
	}
	return mostRecentReversedAt
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

	// The day before Valve added the ability to reverse trades
	mostRecent := uint64(time.Date(2025, 7, 14, 0, 0, 0, 0, time.UTC).UnixMilli())
	for _, reversal := range reversals {
		mostRecent = max(mostRecent, reversal.ReversedAt)

		if _, ok := i.cachedSteamIDs[reversal.SteamID]; ok {
			continue
		}
		i.cachedSteamIDs[reversal.SteamID] = struct{}{}
	}

	startTime := time.UnixMilli(int64(mostRecent))
	for {
		select {
		case <-i.ctx.Done():
			return nil
		default:
		}

		endTime := startTime.Add(24 * time.Hour)
		if endTime.After(time.Now()) {
			endTime = time.Now()
		}

		warnings, err := i.fetch(startTime, endTime)
		if err != nil {
			i.log.Errorf("failed to fetch warnings with startTime %v and endTime %v: %v", startTime, endTime, err)
			return fmt.Errorf("failed to fetch warnings with startTime %v and endTime %v: %v", startTime, endTime, err)
		}

		newStartTime := endTime
		mostRecentReversedAt := i.process(warnings)
		if !mostRecentReversedAt.IsZero() {
			newStartTime = mostRecentReversedAt
		}

		// Delay syncing if newStartTime is close to the current time
		if newStartTime.After(time.Now().Add(-30 * time.Minute)) {
			select {
			case <-time.After(30 * time.Minute):
			case <-i.ctx.Done():
				return nil
			}
		}
		startTime = newStartTime
	}
}

func (i *csfloatIngestor) Start() {
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
	i.log.Infof("Stopping csfloatIngestor")
	i.cancel()
	<-i.stopped
}
