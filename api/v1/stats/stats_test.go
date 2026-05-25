package stats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/util"

	"gorm.io/gorm"
)

func resetCache() {
	cache = sync.Map{}
}

func buildHandlerStack(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}

	router := Router()
	finalHandler := middleware.FactoryMiddleware(f)(router)
	return finalHandler, db
}

func TestSummaryHandler(t *testing.T) {
	resetCache()

	handler, db := buildHandlerStack(t)

	now := uint64(time.Now().UnixMilli())
	hourMs := uint64(60 * 60 * 1000)
	withinDay := now - hourMs
	olderThanDay := now - 36*hourMs

	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: withinDay},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(now - 24*hourMs),
		},
	)

	r := httptest.NewRequest(http.MethodGet, "/summary", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got dto.SummaryStats
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := dto.SummaryStats{
		TradersIndexed:    3,
		TradersFlagged:    2,
		TradersFlagged24h: 1,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSummaryHandler_Caches(t *testing.T) {
	resetCache()

	handler, db := buildHandlerStack(t)

	now := uint64(time.Now().UnixMilli())
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: now - 60*60*1000},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
	)

	hit := func() dto.SummaryStats {
		r := httptest.NewRequest(http.MethodGet, "/summary", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		var s dto.SummaryStats
		if err := json.NewDecoder(w.Result().Body).Decode(&s); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return s
	}

	first := hit()

	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: now - 30*60*1000},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
		},
	)

	second := hit()
	if second != first {
		t.Errorf("expected cached response unchanged: first=%+v second=%+v", first, second)
	}
}

func TestDailyHandler(t *testing.T) {
	resetCache()

	handler, db := buildHandlerStack(t)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
			ReversedAt:      uint64(today.UnixMilli()) + 1,
		},
		&models.Reversal{
			Model:           models.Model{ID: 2},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
			ReversedAt:      uint64(today.AddDate(0, 0, -1).Add(12 * time.Hour).UnixMilli()),
		},
	)

	r := httptest.NewRequest(http.MethodGet, "/reversals/daily?days=30", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got dailyResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 30 {
		t.Fatalf("len(data) = %d, want 30", len(got.Data))
	}

	todayKey := today.Format("2006-01-02")
	yesterdayKey := today.AddDate(0, 0, -1).Format("2006-01-02")

	byDate := make(map[string]uint64, len(got.Data))
	for _, b := range got.Data {
		byDate[b.Date] = b.Count
	}
	if byDate[todayKey] != 1 {
		t.Errorf("today bucket = %d, want 1", byDate[todayKey])
	}
	if byDate[yesterdayKey] != 1 {
		t.Errorf("yesterday bucket = %d, want 1", byDate[yesterdayKey])
	}
}

func TestDailyHandler_InvalidDays(t *testing.T) {
	resetCache()
	handler, _ := buildHandlerStack(t)

	testCases := []struct {
		name string
		days string
	}{
		{name: "outOfRange", days: "7"},
		{name: "negative", days: "-1"},
		{name: "nonNumeric", days: "abc"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/reversals/daily?days="+tc.days, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			resp := w.Result()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
			var body errors.Error
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Details != "days must be one of 30, 60, 90" {
				t.Errorf("details = %q, want %q", body.Details, "days must be one of 30, 60, 90")
			}
		})
	}
}

func TestDailyHandler_DefaultDays(t *testing.T) {
	resetCache()
	handler, _ := buildHandlerStack(t)

	r := httptest.NewRequest(http.MethodGet, "/reversals/daily", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var got dailyResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Data) != 30 {
		t.Errorf("default len(data) = %d, want 30", len(got.Data))
	}
}
