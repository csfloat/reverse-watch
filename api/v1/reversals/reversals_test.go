package reversals

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/internal/testutil"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/util"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestCreateReversal(t *testing.T) {
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
		setup        func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error)
		validateFunc func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response)
	}{
		{
			name: "validSingleReversal",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				testKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, nil, err
				}

				id, err := testKey.ID()
				if err != nil {
					return nil, nil, err
				}

				key := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, key)

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

				formattedKey, err := testKey.Format()
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
					t.Errorf("got status code %d, wanted %d", resp.StatusCode, http.StatusOK)
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
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				testKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, nil, err
				}

				id, err := testKey.ID()
				if err != nil {
					return nil, nil, err
				}

				key := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, key)

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

				formattedKey, err := testKey.Format()
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
					t.Errorf("got status code %d, wanted %d", resp.StatusCode, http.StatusOK)
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
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, nil, err
				}

				id, err := secretKey.ID()
				if err != nil {
					return nil, nil, err
				}

				key := &models.Key{
					ID:              id,
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionExport,
				}
				testutil.Insert(t, db, key)

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, nil, err
				}

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
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, []*reversal, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, nil, err
				}

				id, err := secretKey.ID()
				if err != nil {
					return nil, nil, err
				}

				key := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, key)

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid")))
				r.Header.Set("Content-Type", "application/json")

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, nil, err
				}

				r.Header.Set("Authorization", "Bearer "+formattedKey)
				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, data []*reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("got status code %d, wanted %d", resp.StatusCode, http.StatusBadRequest)
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
			r, data, err := tc.setup(db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, data, w.Result())
		})
	}
}
