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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
			factory := testutil.NewTestFactoryWithDB(t, db).WithKey(keyRepo)

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
			factory := testutil.NewTestFactoryWithDB(t, db).WithKey(keyRepo)

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

func TestListKeys(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := private.NewKeyRepository(db, keygen)
	factory := testutil.NewTestFactoryWithDB(t, db).WithKey(keyRepo)

	testMarketplace1 := &models.Marketplace{
		Slug:     "test-marketplace-1",
		Name:     "Test Marketplace 1",
		IsActive: true,
	}
	testMarketplace2 := &models.Marketplace{
		Slug:     "test-marketplace-2",
		Name:     "Test Marketplace 2",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace1, testMarketplace2)

	// Create auth key
	secretKey, err := keygen.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey(): %v", err)
	}

	id, err := secretKey.ID()
	if err != nil {
		t.Fatalf("ID(): %v", err)
	}

	authKey := &models.Key{
		ID:              id,
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace1.Slug,
		Permissions:     models.PermissionManage,
	}
	testutil.Insert(t, db, authKey)

	// Create additional keys for the same marketplace
	key1 := &models.Key{
		ID:              "key-1",
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace1.Slug,
		Permissions:     models.PermissionRead,
	}
	key2 := &models.Key{
		ID:              "key-2",
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace1.Slug,
		Permissions:     models.PermissionWrite,
	}
	testutil.Insert(t, db, key1, key2)

	// Create additional keys for different marketplace
	key3 := &models.Key{
		ID:              "key-3",
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace2.Slug,
		Permissions:     models.PermissionExport,
	}
	testutil.Insert(t, db, key3)

	wantKeys := []*models.Key{key2, key1, authKey}
	wantStatusCode := http.StatusOK

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	formattedKey, err := secretKey.Format()
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}
	r.Header.Set("Authorization", "Bearer "+formattedKey)

	factoryMiddleware := middleware.FactoryMiddleware(factory)
	permissionsMiddleware := middleware.RequirePermissions(models.PermissionManage)
	handler := http.HandlerFunc(listKeys)

	finalHandler := factoryMiddleware(
		middleware.AuthMiddleware(
			permissionsMiddleware(handler),
		),
	)

	w := httptest.NewRecorder()
	finalHandler.ServeHTTP(w, r)

	if w.Code != wantStatusCode {
		t.Errorf("got status code %d, wanted %d", w.Code, wantStatusCode)
	}

	resp := w.Result()
	defer resp.Body.Close()

	var data struct {
		Data []*models.Key `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(wantKeys, data.Data, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Fatal(diff)
	}
}

func TestListKeys_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		setup          func(db *gorm.DB, factory repository.Factory) (*http.Request, error)
		wantStatusCode int
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				return httptest.NewRequest(http.MethodGet, "/", nil), nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "missingKeyFromContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, factory)
				return r.WithContext(ctx), nil
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "failedToListKeys",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				r := httptest.NewRequest(http.MethodGet, "/", nil)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, factory)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, &models.Key{
					MarketplaceSlug: "test-marketplace",
				})

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
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
			keyRepo := private.NewKeyRepository(db, keygen)
			factory := testutil.NewTestFactoryWithDB(t, db).WithKey(keyRepo)

			w := httptest.NewRecorder()
			r, err := tc.setup(db, factory)
			if err != nil {
				t.Fatal(err)
			}

			handler := http.HandlerFunc(listKeys)
			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}
		})
	}
}
