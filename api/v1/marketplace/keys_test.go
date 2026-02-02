package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/internal/testutil"
	"reverse-watch/middleware"
	"reverse-watch/repository/private"
	"reverse-watch/secret"

	"gorm.io/gorm"
)

func TestCreateKey(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		setup          func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error)
		wantStatusCode int
		wantRawKey     bool
	}{
		{
			name: "validRequest",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				body := struct {
					Permissions models.Permissions `json:"permissions"`
				}{
					Permissions: models.PermissionWrite,
				}

				payload, err := json.Marshal(body)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, err
				}

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, err
				}
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				id, err := secretKey.ID()
				if err != nil {
					return nil, err
				}
				testKey := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionManage,
				}
				testutil.Insert(t, db, testKey)
				return r, nil
			},
			wantStatusCode: http.StatusOK,
			wantRawKey:     true,
		},
		{
			name: "invalidBody",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				body := `{"permissions": 1}`
				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(body)))
				r.Header.Set("Content-Type", "application/json")

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, err
				}

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, err
				}
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				id, err := secretKey.ID()
				if err != nil {
					return nil, err
				}
				testKey := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionManage,
				}
				testutil.Insert(t, db, testKey)
				return r, nil
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalidPermissions",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				body := struct {
					Permissions models.Permissions `json:"permissions"`
				}{
					Permissions: 0,
				}

				payload, err := json.Marshal(body)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, err
				}

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, err
				}
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				id, err := secretKey.ID()
				if err != nil {
					return nil, err
				}
				testKey := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionManage,
				}
				testutil.Insert(t, db, testKey)
				return r, nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantRawKey:     false,
		},
		{
			name: "adminKeyNonCSFloat",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				body := struct {
					Permissions models.Permissions `json:"permissions"`
				}{
					Permissions: models.PermissionAdmin,
				}

				payload, err := json.Marshal(body)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")

				secretKey, err := keygen.GenerateSecretKey()
				if err != nil {
					return nil, err
				}

				formattedKey, err := secretKey.Format()
				if err != nil {
					return nil, err
				}
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				id, err := secretKey.ID()
				if err != nil {
					return nil, err
				}
				testKey := &models.Key{
					ID:              id,
					Environment:     keygen.Environment(),
					MarketplaceSlug: testMarketplace.Slug,
					Permissions:     models.PermissionManage,
				}
				testutil.Insert(t, db, testKey)
				return r, nil
			},
			wantStatusCode: http.StatusBadRequest,
			wantRawKey:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
			keyRepo := private.NewKeyRepository(db, keygen)
			factory := testutil.NewTestFactory(t).WithKey(keyRepo)

			factoryMiddleware := middleware.FactoryMiddleware(factory)
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionManage)
			handler := http.HandlerFunc(createKey)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, err := tc.setup(db, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}

			if tc.wantRawKey {
				resp := w.Result()

				var rawKey *dto.RawKey
				defer resp.Body.Close()
				if err := json.NewDecoder(resp.Body).Decode(&rawKey); err != nil {
					t.Fatal(err)
				}

				if rawKey == nil {
					t.Fatal("got nil RawKey, wanted non-nil")
				}
			}
		})
	}
}

func TestCreateKey_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		setup          func(db *gorm.DB, factory repository.Factory) (*http.Request, error)
		wantStatusCode int
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				return httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(nil)), nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "missingKeyFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(nil))
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, factory)
				return r.WithContext(ctx), nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "invalidSlug",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				body := struct {
					Permissions models.Permissions `json:"permissions"`
				}{
					Permissions: models.PermissionRead,
				}

				payload, err := json.Marshal(body)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")

				testKey := &models.Key{
					ID:              "test-key",
					MarketplaceSlug: "test-marketplace",
					Environment:     constants.EnvironmentDevelopment,
					Permissions:     models.PermissionWrite,
				}
				ctx := context.WithValue(r.Context(), middleware.KeyContextKey, testKey)
				ctx = context.WithValue(ctx, middleware.FactoryContextKey, factory)
				return r.WithContext(ctx), nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
			keyRepo := private.NewKeyRepository(db, keygen)
			factory := testutil.NewTestFactory(t).WithKey(keyRepo)

			w := httptest.NewRecorder()
			r, err := tc.setup(db, factory)
			if err != nil {
				t.Fatal(err)
			}

			handler := http.HandlerFunc(createKey)
			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}
		})
	}
}
