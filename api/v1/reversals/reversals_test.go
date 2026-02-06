package reversals

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
	"reverse-watch/util"

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
