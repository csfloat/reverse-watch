package reversals

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"testing"
	"time"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/util"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestCreateReversals(t *testing.T) {
	t.Parallel()

	type reversal struct {
		SteamID        models.SteamID  `json:"steam_id"`
		Source         *models.Source  `json:"source"`
		RelatedSteamID *models.SteamID `json:"related_steam_id"`
		ReversedAt     uint64          `json:"reversed_at"`
	}

	type req struct {
		Data []*reversal `json:"data"`
	}

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error)
		validateFunc func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response)
	}{
		{
			name: "validSingleReversal",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				data := []*reversal{
					{
						SteamID:        models.SteamID(76561197960287930),
						Source:         util.Ptr(models.SourceRelatedUser),
						RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
						ReversedAt:     1717756800,
					},
				}
				body, err := json.Marshal(&req{Data: data})
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, data, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData []*models.Reversal
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				reversals := make([]*models.Reversal, 0)
				for _, reversal := range data {
					reversals = append(reversals, &models.Reversal{
						SteamID:         reversal.SteamID,
						MarketplaceSlug: "test-marketplace",
						Source:          reversal.Source,
						RelatedSteamID:  reversal.RelatedSteamID,
						ReversedAt:      reversal.ReversedAt,
					})
				}

				if diff := cmp.Diff(respData, reversals, cmpopts.IgnoreFields(models.Reversal{}, "ID", "CreatedAt", "UpdatedAt", "MarketplaceSlug")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "multipleReversals",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				data := []*reversal{
					{
						SteamID:        models.SteamID(76561197960287930),
						Source:         util.Ptr(models.SourceRelatedUser),
						RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
						ReversedAt:     1717756800,
					},
					{
						SteamID:    models.SteamID(76561197960287931),
						ReversedAt: 1717756801,
					},
					{
						SteamID:    models.SteamID(76561197960287932),
						Source:     util.Ptr(models.SourceDirect),
						ReversedAt: 1717756802,
					},
					{
						SteamID:    models.SteamID(76561197960287933),
						Source:     util.Ptr(models.SourceUserReport),
						ReversedAt: 1717756803,
					},
				}
				body, err := json.Marshal(&req{Data: data})
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, data, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData []*models.Reversal
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				reversals := make([]*models.Reversal, 0)
				for _, reversal := range data {
					reversals = append(reversals, &models.Reversal{
						SteamID:         reversal.SteamID,
						MarketplaceSlug: "test-marketplace",
						Source:          reversal.Source,
						RelatedSteamID:  reversal.RelatedSteamID,
						ReversedAt:      reversal.ReversedAt,
					})
				}

				if diff := cmp.Diff(respData, reversals, cmpopts.IgnoreFields(models.Reversal{}, "ID", "CreatedAt", "UpdatedAt", "MarketplaceSlug")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("{}")))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}
			},
		},
		{
			name: "invalidRequestBody",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid")))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}
			},
		},
		{
			name: "createFailedInvalidReversal",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				data := []*reversal{
					{
						SteamID:        models.SteamID(76561197960287930),
						Source:         util.Ptr(models.SourceUserReport),
						RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
						ReversedAt:     1717756800,
					},
				}
				body, err := json.Marshal(&req{Data: data})
				if err != nil {
					return nil, nil, err
				}
				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(body))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "failed to create reversals" {
					t.Errorf("wanted details %q, got %q", "failed to create reversals", respData.Details)
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionWrite)
			handler := http.HandlerFunc(createReversals)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, data, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, data, w.Result())
		})
	}
}

func TestCreateReversals_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(f repository.Factory) *http.Request
		validateFunc func(t *testing.T, resp *http.Response)
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(f repository.Factory) *http.Request {
				return httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("{}")))
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()

				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.InternalServerError, "missing factory from context")
				if diff := cmp.Diff(wantErr, &respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "missingKeyFromContext",
			setup: func(f repository.Factory) *http.Request {
				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("{}")))
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				return r.WithContext(ctx)
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.InternalServerError, "missing key from context")
				if diff := cmp.Diff(wantErr, &respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			f, err := factory.NewFactoryWithConfig(&factory.Config{
				PrivateDB: db,
				PublicDB:  db,
				KeyGen:    secret.NewKeyGenerator(constants.EnvironmentDevelopment),
			})
			if err != nil {
				t.Fatalf("NewFactoryWithConfig(): %v", err)
			}

			w := httptest.NewRecorder()
			r := tc.setup(f)

			handler := http.HandlerFunc(createReversals)
			handler.ServeHTTP(w, r)

			tc.validateFunc(t, w.Result())
		})
	}
}

func generateReversals(t *testing.T, n int, slug string) []*models.Reversal {
	t.Helper()

	reversals := make([]*models.Reversal, 0)
	id := models.Snowflake(1)
	for i := 0; i < n; i++ {
		reversals = append(reversals, &models.Reversal{
			Model: models.Model{
				ID: id,
			},
			SteamID:         models.SteamID(76561197960287930 + uint64(i)),
			MarketplaceSlug: slug,
		})
		id++
	}
	return reversals
}

func TestListReversals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error)
		validateFunc func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response)
	}{
		{
			name: "validListWithDefaults",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 5001, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				wantResponse := &listReversalsResponse{
					Data: reversals[:5000],
					Metadata: metadata{
						Count: 5000,
						NextCursor: &dto.Cursor{
							ID: reversals[4999].ID,
						},
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "filterBySteamID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)
				otherMarketplace := &models.Marketplace{
					Slug:     "other-marketplace",
					Name:     "Other Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, otherMarketplace)

				// Create reversals with different steam IDs
				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         models.SteamID(76561197960287931),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: otherMarketplace.Slug,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/?steam_id=76561197960287930", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				// Only reversals with the target steam ID, sorted by ID DESC
				expectedReversals := []*models.Reversal{reversals[2], reversals[0]}

				wantResponse := &listReversalsResponse{
					Data: expectedReversals,
					Metadata: metadata{
						Count:      2,
						NextCursor: nil,
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "filterByMarketplaceSlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				// Create another marketplace
				otherMarketplace := &models.Marketplace{
					Slug:     "other-marketplace",
					Name:     "Other Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, otherMarketplace)

				// Create reversals for both marketplaces
				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         models.SteamID(76561197960287931),
						MarketplaceSlug: otherMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         models.SteamID(76561197960287932),
						MarketplaceSlug: testMarketplace.Slug,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/?marketplace_slug="+testMarketplace.Slug, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				// Only reversals from test marketplace, sorted by ID desc
				expectedReversals := []*models.Reversal{reversals[2], reversals[0]}

				wantResponse := &listReversalsResponse{
					Data: expectedReversals,
					Metadata: metadata{
						Count:      2,
						NextCursor: nil,
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "customLimit",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 10, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				r := httptest.NewRequest(http.MethodGet, "/?limit=5", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				wantResponse := &listReversalsResponse{
					Data: reversals[:5],
					Metadata: metadata{
						Count: 5,
						NextCursor: &dto.Cursor{
							ID: reversals[4].ID,
						},
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "paginationWithCursor",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 10, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				// Create cursor for second page
				cursor := &dto.Cursor{ID: reversals[4].ID}
				encodedCursor, err := cursor.Encode()
				if err != nil {
					return nil, nil, err
				}

				target := fmt.Sprintf("/?limit=5&cursor=%s", *encodedCursor)
				r := httptest.NewRequest(http.MethodGet, target, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				wantResponse := &listReversalsResponse{
					Data: reversals[5:],
					Metadata: metadata{
						Count: 5,
						NextCursor: &dto.Cursor{
							ID: reversals[9].ID, // Last item in the page
						},
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidSteamID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?steam_id=invalid", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
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
			name: "invalidLimit",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid limit" {
					t.Errorf("wanted details %q, got %q", "invalid limit", respData.Details)
				}
			},
		},
		{
			name: "limitExceedsMax",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?limit=15000", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "limit exceeds max limit of 10000" {
					t.Errorf("wanted details %q, got %q", "limit exceeds max limit of 10000", respData.Details)
				}
			},
		},
		{
			name: "invalidCursor",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?cursor=invalid-base64", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid cursor" {
					t.Errorf("wanted details %q, got %q", "invalid cursor", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}
			},
		},
		{
			name: "emptyResults",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *listReversalsResponse, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				wantResponse := &listReversalsResponse{
					Data: []*models.Reversal{},
					Metadata: metadata{
						Count:      0,
						NextCursor: nil,
					},
				}
				return r, wantResponse, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData); diff != "" {
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionExport)
			handler := http.HandlerFunc(listReversalsHandler)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, expectedResult, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, expectedResult, w.Result())
		})
	}
}

func TestListReversals_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(db *gorm.DB, factory repository.Factory) (*http.Request, error)
		validateFunc func(t *testing.T, resp *http.Response)
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				return httptest.NewRequest(http.MethodGet, "/", nil), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()

				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.InternalServerError, "missing factory from context")
				if diff := cmp.Diff(wantErr, &respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "failedToListReversals",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, factory)

				// Close db connection to simulate database error
				sqlDb, err := db.DB()
				if err != nil {
					return nil, err
				}
				if err := sqlDb.Close(); err != nil {
					return nil, err
				}
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()

				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "failed to list reversals" {
					t.Errorf("wanted details %q, got %q", "failed to list reversals", respData.Details)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			f, err := factory.NewFactoryWithConfig(&factory.Config{
				PrivateDB: db,
				PublicDB:  db,
				KeyGen:    secret.NewKeyGenerator(constants.EnvironmentDevelopment),
			})
			if err != nil {
				t.Fatalf("NewFactoryWithConfig(): %v", err)
			}

			w := httptest.NewRecorder()
			r, err := tc.setup(db, f)
			if err != nil {
				t.Fatal(err)
			}

			handler := http.HandlerFunc(listReversalsHandler)
			handler.ServeHTTP(w, r)

			tc.validateFunc(t, w.Result())
		})
	}
}

func TestListReversals_Pagination(t *testing.T) {
	t.Parallel()

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

	testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

	reversals := generateReversals(t, 200, testMarketplace.Slug)
	testutil.Insert(t, db, reversals...)

	sort.Slice(reversals, func(i, j int) bool {
		return reversals[i].ID > reversals[j].ID
	})

	testCases := []struct {
		name     string
		setup    func(t *testing.T) (*http.Request, *listReversalsResponse, error)
		validate func(t *testing.T, expectedResult *listReversalsResponse, resp *http.Response)
	}{
		{
			name: "firstPageLimit75",
			setup: func(t *testing.T) (*http.Request, *listReversalsResponse, error) {
				r := httptest.NewRequest(http.MethodGet, "/?limit=75", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				expectedResult := &listReversalsResponse{
					Data: reversals[:75],
					Metadata: metadata{
						Count: 75,
						NextCursor: &dto.Cursor{
							ID: reversals[74].ID,
						},
					},
				}
				return r, expectedResult, nil
			},
			validate: func(t *testing.T, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "secondPageLimit50",
			setup: func(t *testing.T) (*http.Request, *listReversalsResponse, error) {
				cursor := &dto.Cursor{
					ID: reversals[74].ID,
				}
				encodedCursor, err := cursor.Encode()
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodGet, "/?limit=50&cursor="+*encodedCursor, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				expectedResult := &listReversalsResponse{
					Data: reversals[75:125],
					Metadata: metadata{
						Count: 50,
						NextCursor: &dto.Cursor{
							ID: reversals[124].ID,
						},
					},
				}
				return r, expectedResult, nil
			},
			validate: func(t *testing.T, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "thirdPageLimit50",
			setup: func(t *testing.T) (*http.Request, *listReversalsResponse, error) {
				cursor := &dto.Cursor{
					ID: reversals[124].ID,
				}
				encodedCursor, err := cursor.Encode()
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodGet, "/?limit=50&cursor="+*encodedCursor, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				expectedResult := &listReversalsResponse{
					Data: reversals[125:175],
					Metadata: metadata{
						Count: 50,
						NextCursor: &dto.Cursor{
							ID: reversals[174].ID,
						},
					},
				}
				return r, expectedResult, nil
			},
			validate: func(t *testing.T, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "lastPageLimit50",
			setup: func(t *testing.T) (*http.Request, *listReversalsResponse, error) {
				cursor := &dto.Cursor{
					ID: reversals[174].ID,
				}
				encodedCursor, err := cursor.Encode()
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodGet, "/?limit=50&cursor="+*encodedCursor, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				expectedResult := &listReversalsResponse{
					Data: reversals[175:200],
					Metadata: metadata{
						Count: 25,
					},
				}
				return r, expectedResult, nil
			},
			validate: func(t *testing.T, expectedResult *listReversalsResponse, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData listReversalsResponse
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(expectedResult, &respData); diff != "" {
					t.Error(diff)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, expectedResult, err := tc.setup(t)
			if err != nil {
				t.Fatalf("setup(): %v", err)
			}

			factoryMiddleware := middleware.FactoryMiddleware(f)
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionExport)
			handler := http.HandlerFunc(listReversalsHandler)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			finalHandler.ServeHTTP(w, r)
			tc.validate(t, expectedResult, w.Result())
		})
	}
}

func decodeExportedCSV(t *testing.T, records [][]string) []*models.Reversal {
	t.Helper()

	headers := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
	reversals := make([]*models.Reversal, 0)

	for i := 0; i < len(records); i++ {
		row := records[i]
		reversal := &models.Reversal{
			Model: models.Model{},
		}
		for j := 0; j < len(row); j++ {
			column := row[j]

			switch headers[j] {
			case "id":
				if column != "" {
					id, err := models.ToSnowflake(column)
					if err != nil {
						t.Fatalf("ToSnowflake(%q): %v", column, err)
					}
					reversal.Model.ID = id
				}
			case "created_at":
				if column != "" {
					timestamp, err := strconv.ParseUint(column, 10, 64)
					if err != nil {
						t.Fatalf("ParseUint(%q): %v", column, err)
					}
					reversal.Model.CreatedAt = timestamp
				}
			case "updated_at":
				if column != "" {
					timestamp, err := strconv.ParseUint(column, 10, 64)
					if err != nil {
						t.Fatalf("ParseUint(%q): %v", column, err)
					}
					reversal.Model.UpdatedAt = timestamp
				}
			case "steam_id":
				if column != "" {
					steamId, err := models.ToSteamID(column)
					if err != nil {
						t.Fatalf("ToSteamID(%q): %v", column, err)
					}
					reversal.SteamID = *steamId
				}
			case "marketplace_slug":
				if column != "" {
					reversal.MarketplaceSlug = column
				}
			case "source":
				if column != "" {
					switch column {
					case "direct":
						reversal.Source = util.Ptr(models.SourceDirect)
					case "related_user":
						reversal.Source = util.Ptr(models.SourceRelatedUser)
					case "user_report":
						reversal.Source = util.Ptr(models.SourceUserReport)
					default:
						t.Fatalf("unknown source column: %q", column)
					}
				}
			case "related_steam_id":
				if column != "" {
					steamId, err := models.ToSteamID(column)
					if err != nil {
						t.Fatalf("ToSteamID(%q): %v", column, err)
					}
					reversal.RelatedSteamID = steamId
				}
			case "reversed_at":
				if column != "" {
					timestamp, err := strconv.ParseUint(column, 10, 64)
					if err != nil {
						t.Fatalf("ParseUint(%q): %v", column, err)
					}
					reversal.ReversedAt = timestamp
				}
			case "expunged_at":
				if column != "" {
					timestamp, err := strconv.ParseUint(column, 10, 64)
					if err != nil {
						t.Fatalf("ParseUint(%q): %v", column, err)
					}
					reversal.ExpungedAt = &timestamp
				}
			default:
				t.Fatalf("unknown column: %q", headers[j])
			}
		}

		reversals = append(reversals, reversal)
	}
	return reversals
}

func TestExportReversals(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error)
		validateFunc func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response)
	}{
		{
			name: "validExportWithDefaults",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 100, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				contentType := resp.Header.Get("Content-Type")
				if contentType != "text/csv; charset=utf-8" {
					t.Errorf("wanted Content-Type %q, got %q", "text/csv; charset=utf-8", contentType)
				}

				nextCursor := resp.Header.Get("X-Next-Cursor")
				if nextCursor != "" {
					t.Error("expected no X-Next-Cursor header for data less than default limit")
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "singleReversalExport",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
						Source:          util.Ptr(models.SourceRelatedUser),
						RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
						ReversedAt:      1717756800,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				if len(records) != 2 {
					t.Fatalf("expected 2 rows (header + 1 data), got %d", len(records))
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "exportWithNullFields",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
						Source:          nil,
						RelatedSteamID:  nil,
						ReversedAt:      1717756800,
						ExpungedAt:      nil,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "customLimit",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 20, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				r := httptest.NewRequest(http.MethodGet, "/?limit=10", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals[:10], nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				nextCursor := resp.Header.Get("X-Next-Cursor")
				if nextCursor == "" {
					t.Error("expected X-Next-Cursor header to be set")
				}

				decodedCursor, err := dto.DecodeCursor(nextCursor)
				if err != nil {
					t.Fatalf("DecodeCursor(%q): %v", nextCursor, err)
				}

				expectedCursor := &dto.Cursor{ID: expectedReversals[9].ID}
				if diff := cmp.Diff(expectedCursor, decodedCursor); diff != "" {
					t.Error(diff)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "filterBySteamID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						Source:          util.Ptr(models.SourceUserReport),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         models.SteamID(76561197960287931),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/?steam_id=76561197960287930", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, []*models.Reversal{reversals[2], reversals[0]}, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "filterByMarketplaceSlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				otherMarketplace := &models.Marketplace{
					Slug:     "other-marketplace",
					Name:     "Other Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, otherMarketplace)

				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         models.SteamID(76561197960287930),
						MarketplaceSlug: testMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         models.SteamID(76561197960287931),
						MarketplaceSlug: otherMarketplace.Slug,
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         models.SteamID(76561197960287932),
						MarketplaceSlug: testMarketplace.Slug,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodGet, "/?marketplace_slug="+testMarketplace.Slug, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, []*models.Reversal{reversals[2], reversals[0]}, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "paginationWithCursor",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 20, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				cursor := &dto.Cursor{ID: reversals[9].ID}
				encodedCursor, err := cursor.Encode()
				if err != nil {
					return nil, nil, err
				}

				target := fmt.Sprintf("/?limit=10&cursor=%s", *encodedCursor)
				r := httptest.NewRequest(http.MethodGet, target, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals[10:], nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "lastPageNoNextCursor",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				reversals := generateReversals(t, 5, testMarketplace.Slug)
				testutil.Insert(t, db, reversals...)

				sort.Slice(reversals, func(i, j int) bool {
					return reversals[i].ID > reversals[j].ID
				})

				r := httptest.NewRequest(http.MethodGet, "/?limit=10", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversals, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				nextCursor := resp.Header.Get("X-Next-Cursor")
				if nextCursor != "" {
					t.Errorf("expected no X-Next-Cursor header on last page, got %q", nextCursor)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
					t.Error(diff)
				}

				exportedReversals := decodeExportedCSV(t, records[1:])
				if diff := cmp.Diff(expectedReversals, exportedReversals); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidSteamID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?steam_id=invalid", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
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
			name: "invalidLimit",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid limit" {
					t.Errorf("wanted details %q, got %q", "invalid limit", respData.Details)
				}
			},
		},
		{
			name: "limitExceedsMax",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?limit=60000", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "limit exceeds max limit of 50000" {
					t.Errorf("wanted details %q, got %q", "limit exceeds max limit of 50000", respData.Details)
				}
			},
		},
		{
			name: "invalidCursor",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/?cursor=invalid-base64", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid cursor" {
					t.Errorf("wanted details %q, got %q", "invalid cursor", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}
			},
		},
		{
			name: "emptyResults",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionExport)

				r := httptest.NewRequest(http.MethodGet, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, expectedReversals []*models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				reader := csv.NewReader(resp.Body)
				records, err := reader.ReadAll()
				if err != nil {
					t.Fatalf("failed to read CSV: %v", err)
				}

				if len(records) != 1 {
					t.Errorf("expected only header row, got %d rows", len(records))
				}

				expectedHeaders := []string{"id", "created_at", "updated_at", "steam_id", "marketplace_slug", "source", "related_steam_id", "reversed_at", "expunged_at"}
				if diff := cmp.Diff(expectedHeaders, records[0]); diff != "" {
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionExport)
			handler := http.HandlerFunc(exportReversals)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, expectedReversals, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, expectedReversals, w.Result())
		})
	}
}

func TestExportReversals_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(db *gorm.DB, factory repository.Factory) (*http.Request, error)
		validateFunc func(t *testing.T, resp *http.Response)
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				return httptest.NewRequest(http.MethodGet, "/", nil), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()

				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.InternalServerError, "missing factory from context")
				if diff := cmp.Diff(wantErr, &respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			f, err := factory.NewFactoryWithConfig(&factory.Config{
				PrivateDB: db,
				PublicDB:  db,
				KeyGen:    secret.NewKeyGenerator(constants.EnvironmentDevelopment),
			})
			if err != nil {
				t.Fatalf("NewFactoryWithConfig(): %v", err)
			}

			w := httptest.NewRecorder()
			r, err := tc.setup(db, f)
			if err != nil {
				t.Fatal(err)
			}

			handler := http.HandlerFunc(exportReversals)
			handler.ServeHTTP(w, r)

			tc.validateFunc(t, w.Result())
		})
	}
}

func TestExpungeReversal(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal)
		validateFunc func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response)
	}{
		{
			name: "validExpunge",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodDelete, "/"+reversal.ID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", reversal.ID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), reversal
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				// Verify reversal was expunged
				var storedReversal models.Reversal
				if err := db.Where("id = ?", reversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.ExpungedAt == nil {
					t.Error("expected reversal to be expunged, but ExpungedAt is nil")
				}
			},
		},
		{
			name: "reversalNotFound",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				nonExistentID := models.Snowflake(999999)
				r := httptest.NewRequest(http.MethodDelete, "/"+nonExistentID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", nonExistentID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "failed to expunge reversal" {
					t.Errorf("wanted details %q, got %q", "failed to expunge reversal", respData.Details)
				}
			},
		},
		{
			name: "reversalAlreadyExpunged",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				createdAt := uint64(time.Now().Add(-1 * time.Minute).UnixMilli())
				reversal := &models.Reversal{
					Model:           models.Model{ID: 1, CreatedAt: createdAt},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-marketplace",
					ReversedAt:      1717756800,
					ExpungedAt:      &createdAt,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodDelete, "/"+reversal.ID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", reversal.ID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), reversal
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "reversal has already been expunged" {
					t.Fatalf("wanted details %q, got %q", "reversal has already been expunged", respData.Details)
				}

				// Verify expunged at was NOT updated
				var storedReversal models.Reversal
				if err := db.Where("id = ?", reversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if *storedReversal.ExpungedAt != *reversal.ExpungedAt {
					t.Errorf("expected expunged at %d, got %d", reversal.ExpungedAt, storedReversal.ExpungedAt)
				}
			},
		},
		{
			name: "cannotExpungeOtherMarketplaceReversal",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				// Create another marketplace
				otherMarketplace := &models.Marketplace{
					Slug:     "other-marketplace",
					Name:     "Other Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, otherMarketplace)

				// Create reversal for other marketplace
				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: otherMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodDelete, "/"+reversal.ID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", reversal.ID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), reversal
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "cannot expunge reversal report of another marketplace" {
					t.Errorf("wanted details %q, got %q", "cannot expunge reversal report of another marketplace", respData.Details)
				}

				// Verify reversal was NOT expunged
				var storedReversal models.Reversal
				if err := db.Where("id = ?", reversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.ExpungedAt != nil {
					t.Error("expected reversal to not be expunged, but ExpungedAt is set")
				}
			},
		},
		{
			name: "invalidID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				r := httptest.NewRequest(http.MethodDelete, "/invalid", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", "invalid")
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid id" {
					t.Errorf("wanted details %q, got %q", "invalid id", respData.Details)
				}
			},
		},
		{
			name: "emptyID",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionDelete)

				r := httptest.NewRequest(http.MethodDelete, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", "")
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "invalid id" {
					t.Errorf("wanted details %q, got %q", "invalid id", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodDelete, "/"+reversal.ID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", reversal.ID.String())
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)

				return r.WithContext(ctx), reversal
			},
			validateFunc: func(t *testing.T, db *gorm.DB, reversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}

				// Verify reversal was NOT expunged
				var storedReversal models.Reversal
				if err := db.Where("id = ?", reversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.ExpungedAt != nil {
					t.Error("expected reversal to not be expunged, but ExpungedAt is set")
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionDelete)
			handler := http.HandlerFunc(expungeReversal)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, reversalID := tc.setup(t, db, f, keygen)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, reversalID, w.Result())
		})
	}
}
