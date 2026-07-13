package stats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
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

	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

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
			ReversedAt:      withinDay,
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
			ReversedAt:      olderThanDay,
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(now - 24*hourMs),
			ReversedAt:      olderThanDay,
		},
	)

	// 2 distinct Steam IDs searched a total of 3 times.
	testutil.Insert(t, db,
		&models.SearchCount{SteamID: models.SteamID(76561197960287940), Count: 2, LastSearchedAt: now},
		&models.SearchCount{SteamID: models.SteamID(76561197960287941), Count: 1, LastSearchedAt: now},
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
		SteamIDsSearched:  2,
		TotalSearches:     3,
		TradersFlagged:    2,
		TradersFlagged24h: 1,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SummaryStats mismatch (-want +got):\n%s", diff)
	}
}

func TestDailyHandler(t *testing.T) {
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
		byDate[b.Date.UTC().Format("2006-01-02")] = b.Count
	}
	if byDate[todayKey] != 1 {
		t.Errorf("today bucket = %d, want 1", byDate[todayKey])
	}
	if byDate[yesterdayKey] != 1 {
		t.Errorf("yesterday bucket = %d, want 1", byDate[yesterdayKey])
	}
}

func TestDailyHandler_InvalidDays(t *testing.T) {
	handler, _ := buildHandlerStack(t)

	const wantDetails = "days must be one of 7, 30, 60, 90, 180, 365"
	testCases := []struct {
		name string
		days string
	}{
		{name: "outOfRange", days: "45"},
		{name: "negative", days: "-1"},
		{name: "nonNumeric", days: "abc"},
		{name: "zero", days: "0"},
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
			if body.Details != wantDetails {
				t.Errorf("details = %q, want %q", body.Details, wantDetails)
			}
		})
	}
}

func TestDailyHandler_AcceptedDays(t *testing.T) {
	// Each accepted value should return a fully zero-filled series of
	// exactly that many buckets. Empty DB keeps the assertion focused on
	// the length contract that the picker depends on.
	for _, days := range []int{7, 30, 60, 90, 180, 365} {
		days := days
		t.Run(strconv.Itoa(days), func(t *testing.T) {
			handler, _ := buildHandlerStack(t)

			r := httptest.NewRequest(http.MethodGet, "/reversals/daily?days="+strconv.Itoa(days), nil)
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
			if len(got.Data) != days {
				t.Errorf("len(data) = %d, want %d", len(got.Data), days)
			}
		})
	}
}

func TestDailyHandler_DefaultDays(t *testing.T) {
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
