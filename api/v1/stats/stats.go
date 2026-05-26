package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/repository"
	"reverse-watch/errors"
	"reverse-watch/middleware"
	"reverse-watch/render"
)

const cacheTTL = 60 * time.Second

var allowedDays = map[int]bool{7: true, 30: true, 60: true, 90: true, 180: true, 365: true}

type cacheEntry struct {
	at      time.Time
	payload []byte
}

var cache sync.Map

func cacheGet(key string) ([]byte, bool) {
	v, ok := cache.Load(key)
	if !ok {
		return nil, false
	}
	e := v.(cacheEntry)
	if time.Since(e.at) > cacheTTL {
		return nil, false
	}
	return e.payload, true
}

func cacheSet(key string, payload []byte) {
	cache.Store(key, cacheEntry{at: time.Now(), payload: payload})
}

func writeCachedJSON(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func summaryHandler(w http.ResponseWriter, r *http.Request) {
	const key = "summary"
	if payload, ok := cacheGet(key); ok {
		writeCachedJSON(w, payload)
		return
	}

	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	stats, err := factory.Reversal().SummaryStats()
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to load summary stats")
		return
	}

	payload, err := json.Marshal(stats)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to encode summary stats")
		return
	}
	cacheSet(key, payload)
	writeCachedJSON(w, payload)
}

type dailyResponse struct {
	Data []dto.DailyCount `json:"data"`
}

func dailyHandler(w http.ResponseWriter, r *http.Request) {
	days := 30
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		parsed, err := strconv.Atoi(daysStr)
		if err != nil || !allowedDays[parsed] {
			render.Errorf(w, r, errors.BadRequest, "days must be one of 7, 30, 60, 90, 180, 365")
			return
		}
		days = parsed
	}

	key := fmt.Sprintf("daily:%d", days)
	if payload, ok := cacheGet(key); ok {
		writeCachedJSON(w, payload)
		return
	}

	factory := r.Context().Value(middleware.FactoryContextKey).(repository.Factory)

	counts, err := factory.Reversal().DailyCounts(days)
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to load daily counts")
		return
	}

	payload, err := json.Marshal(dailyResponse{Data: counts})
	if err != nil {
		render.Errorf(w, r, errors.InternalServerError, "failed to encode daily counts")
		return
	}
	cacheSet(key, payload)
	writeCachedJSON(w, payload)
}
