package reversals

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func buildRecentHandlerStack(t *testing.T) (http.Handler, *gorm.DB) {
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

	handler := http.HandlerFunc(listRecentHandler)
	return middleware.FactoryMiddleware(f)(handler), db
}

func TestListRecentHandler(t *testing.T) {
	t.Parallel()

	handler, db := buildRecentHandlerStack(t)

	base := models.Epoch + 1000

	// 5 rows, monotonically increasing CreatedAt. Row id=3 is expunged.
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: base + 100},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: base + 200},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: base + 300},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(base + 400),
		},
		&models.Reversal{
			Model:           models.Model{ID: 4, CreatedAt: base + 500},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 5, CreatedAt: base + 600},
			SteamID:         models.SteamID(76561197960287934),
			MarketplaceSlug: "csfloat",
		},
	)

	r := httptest.NewRequest(http.MethodGet, "/recent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var body listRecentResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	wantSteamIDs := []models.SteamID{
		76561197960287934, // id=5
		76561197960287933, // id=4
		76561197960287931, // id=2
		76561197960287930, // id=1
	}
	if len(body.Data) != len(wantSteamIDs) {
		t.Fatalf("len(data) = %d, want %d", len(body.Data), len(wantSteamIDs))
	}
	for i, want := range wantSteamIDs {
		if body.Data[i].SteamID != want {
			t.Errorf("data[%d].SteamID = %d, want %d", i, body.Data[i].SteamID, want)
		}
	}
}

func TestListRecentHandler_RespectsLimit(t *testing.T) {
	t.Parallel()

	handler, db := buildRecentHandlerStack(t)

	base := models.Epoch + 1000
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: base + 100},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: base + 200},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: base + 300},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
		},
	)

	r := httptest.NewRequest(http.MethodGet, "/recent?limit=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body listRecentResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Data) != 2 {
		t.Errorf("len(data) = %d, want 2", len(body.Data))
	}
}

func TestListRecentHandler_InvalidLimit(t *testing.T) {
	t.Parallel()

	handler, _ := buildRecentHandlerStack(t)

	testCases := []struct {
		name  string
		limit string
	}{
		{name: "zero", limit: "0"},
		{name: "negative", limit: "-1"},
		{name: "overMax", limit: "101"},
		{name: "nonNumeric", limit: "abc"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/recent?limit="+tc.limit, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			resp := w.Result()
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
			}
			var body errors.Error
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body.Details != "limit must be between 1 and 100" {
				t.Errorf("details = %q, want %q", body.Details, "limit must be between 1 and 100")
			}
		})
	}
}

func TestListRecentHandler_ResponseShape(t *testing.T) {
	t.Parallel()

	handler, db := buildRecentHandlerStack(t)

	base := models.Epoch + 1000
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: base + 100},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
			ReversedAt:      base + 50,
		},
	)

	r := httptest.NewRequest(http.MethodGet, "/recent", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// Decode as raw JSON to assert the exact wire shape (especially steam_id as a string).
	var raw struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(w.Result().Body).Decode(&raw); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(raw.Data) != 1 {
		t.Fatalf("len(data) = %d, want 1", len(raw.Data))
	}
	row := raw.Data[0]

	expectedKeys := []string{"marketplace_slug", "steam_id", "reversed_at", "created_at"}
	for _, k := range expectedKeys {
		if _, ok := row[k]; !ok {
			t.Errorf("missing key %q in response", k)
		}
	}
	for k := range row {
		ok := false
		for _, want := range expectedKeys {
			if k == want {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("unexpected key %q in response", k)
		}
	}

	steamIDValue, ok := row["steam_id"].(string)
	if !ok {
		t.Errorf("steam_id should be a JSON string, got %T", row["steam_id"])
	}
	if steamIDValue != "76561197960287930" {
		t.Errorf("steam_id = %q, want %q", steamIDValue, "76561197960287930")
	}
}
