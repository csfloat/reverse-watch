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
					IsExpunged:            false,
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
					IsExpunged:            false,
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
					IsExpunged:            false,
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
					SteamID:               steamID,
					HasReversed:           true,
					IsExpunged:            true,
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

				// Should use most recent reversal (ID 2), which is not expunged
				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					IsExpunged:            false,
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
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
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
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
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

				// Should return all reversals across marketplaces, most recent first
				expectedResp := &fetchUserStatusResponse{
					SteamID:               steamID,
					HasReversed:           true,
					IsExpunged:            false,
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
