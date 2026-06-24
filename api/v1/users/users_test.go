package users

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/util"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"gorm.io/gorm"
)

func TestFetchUserStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse)
		validateFunc func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response)
	}{
		{
			name: "userWithNoReversals",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           false,
					LastReversalTimestamp: nil,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "userWithSingleReversal",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         steamID,
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
					Source:          util.Ptr(models.SourceDirect),
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					LastReversalTimestamp: &reversal.ReversedAt,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "userWithMultipleReversals",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)
				testMarketplaces := []*models.Marketplace{
					{
						Slug:     "test-marketplace-1",
						Name:     "Test Marketplace 1",
						IsActive: true,
					},
					{
						Slug:     "test-marketplace-2",
						Name:     "Test Marketplace 2",
						IsActive: true,
					},
					{
						Slug:     "test-marketplace-3",
						Name:     "Test Marketplace 3",
						IsActive: true,
					},
				}
				testutil.Insert(t, db, testMarketplaces...)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplaces[0].Slug,
						ReversedAt:      1717756800,
						Source:          util.Ptr(models.SourceDirect),
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplaces[1].Slug,
						ReversedAt:      1717756900,
						Source:          util.Ptr(models.SourceUserReport),
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplaces[2].Slug,
						ReversedAt:      1717757000,
						Source:          util.Ptr(models.SourceRelatedUser),
						RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				// Should return the most recent reversal (highest ID)
				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					LastReversalTimestamp: &reversals[2].ReversedAt,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "userWithExpungedReversal",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				expungedAt := uint64(1717756900)
				reversal := &models.Reversal{
					Model:           models.Model{ID: 1, CreatedAt: 1},
					SteamID:         steamID,
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
					ExpungedAt:      &expungedAt,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				expectedResp := &fetchUserStatusResponse{
					SteamID:     steamID,
					HasReversed: false,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "userWithExpungedAndNonExpungedReversals",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)
				testMarketplaces := []*models.Marketplace{
					{
						Slug:     "test-marketplace-1",
						Name:     "Test Marketplace 1",
						IsActive: true,
					},
					{
						Slug:     "test-marketplace-2",
						Name:     "Test Marketplace 2",
						IsActive: true,
					},
				}
				testutil.Insert(t, db, testMarketplaces...)

				expungedAt := uint64(1717756900)
				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1, CreatedAt: 1},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplaces[0].Slug,
						ReversedAt:      1717756800,
						ExpungedAt:      &expungedAt,
					},
					{
						Model:           models.Model{ID: 2, CreatedAt: 1},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplaces[1].Slug,
						ReversedAt:      1717756900,
						ExpungedAt:      nil, // Not expunged
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				// User is considered to have reversed if not all reversal reports have been expunged
				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					LastReversalTimestamp: &reversals[1].ReversedAt,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidSteamID",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				r := httptest.NewRequest(http.MethodGet, "/invalid-steam-id", nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", "invalid-steam-id")
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid steam id" {
					t.Errorf("wanted details %q, got %q", "invalid steam id", respData.Details)
				}
			},
		},
		{
			name: "emptySteamID",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", "")
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid steam id" {
					t.Errorf("wanted details %q, got %q", "invalid steam id", respData.Details)
				}
			},
		},
		{
			name: "userFromMultipleMarketplaces",
			setup: func(t *testing.T, db *gorm.DB) (*http.Request, *fetchUserStatusResponse) {
				steamID := models.SteamID(76561197960287930)
				marketplace1 := &models.Marketplace{
					Slug:     "marketplace-1",
					Name:     "Marketplace 1",
					IsActive: true,
				}
				marketplace2 := &models.Marketplace{
					Slug:     "marketplace-2",
					Name:     "Marketplace 2",
					IsActive: true,
				}
				testutil.Insert(t, db, marketplace1, marketplace2)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         steamID,
						MarketplaceSlug: marketplace1.Slug,
						ReversedAt:      1717756800,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         steamID,
						MarketplaceSlug: marketplace2.Slug,
						ReversedAt:      1717756900,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("steamId", steamID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					LastReversalTimestamp: &reversals[1].ReversedAt,
				}

				return r.WithContext(ctx), expectedResp
			},
			validateFunc: func(t *testing.T, expectedResp *fetchUserStatusResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData fetchUserStatusResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResp, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			factoryMiddleware := middleware.FactoryMiddleware(f)
			handler := http.HandlerFunc(fetchUserStatus)

			finalHandler := factoryMiddleware(handler)

			w := httptest.NewRecorder()
			r, expectedResp := tc.setup(t, db)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, expectedResp, w.Result())
		})
	}
}

func newUserStatusHandler(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()

	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}
	return middleware.FactoryMiddleware(f)(http.HandlerFunc(fetchUserStatus))
}

func userStatusRequest(steamID models.SteamID) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/"+steamID.String(), nil)
	chiContext := chi.NewRouteContext()
	chiContext.URLParams.Add("steamId", steamID.String())
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiContext))
}

func TestFetchUserStatus_IncrementsSearchCount(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	handler := newUserStatusHandler(t, db)
	steamID := models.SteamID(76561197960287930)

	do := func() {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, userStatusRequest(steamID))
		if w.Result().StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Result().StatusCode, http.StatusOK)
		}
	}

	do()
	var sc models.SearchCount
	if err := db.Where("steam_id = ?", uint64(steamID)).First(&sc).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}
	if sc.Count != 1 {
		t.Errorf("Count = %d, want 1", sc.Count)
	}
	if sc.LastSearchedAt == 0 {
		t.Errorf("LastSearchedAt = 0, want non-zero")
	}

	do()
	if err := db.Where("steam_id = ?", uint64(steamID)).First(&sc).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}
	if sc.Count != 2 {
		t.Errorf("Count = %d, want 2", sc.Count)
	}
}

func TestFetchUserStatus_IncrementFailureDoesNotFailLookup(t *testing.T) {
	// A counting failure must never break the user-facing lookup. Drop the
	// search_counts table so the increment errors, then assert the lookup still
	// succeeds. logging.Initialize() is required because the handler logs the
	// swallowed error.
	logging.Initialize()

	db := testutil.NewTestDB(t)
	handler := newUserStatusHandler(t, db)

	if err := db.Migrator().DropTable(&models.SearchCount{}); err != nil {
		t.Fatalf("DropTable(): %v", err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, userStatusRequest(models.SteamID(76561197960287930)))

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d (lookup must succeed despite count failure)", resp.StatusCode, http.StatusOK)
	}

	defer resp.Body.Close()
	var respData fetchUserStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if respData.HasReversed {
		t.Errorf("HasReversed = true, want false")
	}
}
