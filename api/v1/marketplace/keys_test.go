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
	"reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/logging"
	"reverse-watch/middleware"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"

	"github.com/go-chi/chi/v5"
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
			f, err := factory.NewFactoryWithConfig(&factory.Config{
				PrivateDB: db,
				PublicDB:  db,
				KeyGen:    keygen,
			})
			if err != nil {
				t.Fatalf("NewFactoryWithConfig(): %v", err)
			}

			factoryMiddleware := middleware.FactoryMiddleware(f)
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
	f, err := factory.NewFactoryWithConfig(&factory.Config{
		PrivateDB: db,
		PublicDB:  db,
		KeyGen:    keygen,
	})
	if err != nil {
		t.Fatalf("NewFactoryWithConfig(): %v", err)
	}

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

	factoryMiddleware := middleware.FactoryMiddleware(f)
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

			handler := http.HandlerFunc(listKeys)
			handler.ServeHTTP(w, r)

			if w.Code != tc.wantStatusCode {
				t.Errorf("got status code %d, wanted %d", w.Code, tc.wantStatusCode)
			}
		})
	}
}

func TestDeleteKey_WithMiddlewares(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, error)
		validateFunc func(t *testing.T, db *gorm.DB, id string, resp *http.Response)
	}{
		{
			name: "successWithAuth",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)
				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)

				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", keyToDelete.ID)
				ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
				return r.WithContext(ctx), keyToDelete.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, id string, resp *http.Response) {
				if resp.StatusCode != http.StatusNoContent {
					t.Errorf("wanted status code %d, got %d", http.StatusNoContent, resp.StatusCode)
				}

				var key models.Key
				if err := db.First(&key, "id = ?", id).Error; err == nil {
					t.Fatal("expected record to be deleted, but it still exists")
				}
			},
		},
		{
			name: "missingID",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)

				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				return r, "", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, id string, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.BadRequest, "invalid id")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionWrite)
				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				return r, "", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, id string, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.Forbidden
				if diff := cmp.Diff(&wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "invalidToken",
			setup: func(db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, "invalid-token")
				return r, "", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, id string, resp *http.Response) {
				if resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("wanted status code %d, got %d", http.StatusUnauthorized, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.InvalidApiKey
				if diff := cmp.Diff(&wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionManage)
			handler := http.HandlerFunc(deleteKey)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, id, err := tc.setup(db, keygen)
			if err != nil {
				t.Fatalf("setup(): %v", err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, id, w.Result())
		})
	}
}

func TestDeleteKey_Errors(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error)
		validateFunc func(t *testing.T, resp *http.Response)
	}{
		{
			name: "missingID",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)
				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, authKey)
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.BadRequest, "invalid id")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "notFound",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, authKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", "key-to-delete")
				ctx = context.WithValue(ctx, chi.RouteCtxKey, chiContext)
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.DBDelete, "failed to delete key")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "doesntOwnKey",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)

				testMarketplace2 := &models.Marketplace{
					Slug:     "test-marketplace-2",
					Name:     "Test Marketplace 2",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace2)

				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: "test-marketplace-2",
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, authKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", keyToDelete.ID)
				ctx = context.WithValue(ctx, chi.RouteCtxKey, chiContext)
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.BadRequest, "marketplace doesn't own key being deleted")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "environmentMismatch",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)
				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     constants.EnvironmentProduction,
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, authKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", keyToDelete.ID)
				ctx = context.WithValue(ctx, chi.RouteCtxKey, chiContext)
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.BadRequest, "cannot delete key from a different environment")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "deleteKeyUsedForAuth",
			setup: func(db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				testMarketplace, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, keygen, models.PermissionManage)
				keyToDelete := &models.Key{
					ID:              "key-to-delete",
					MarketplaceSlug: testMarketplace.Slug,
					Environment:     keygen.Environment(),
					Permissions:     models.PermissionWrite,
				}
				testutil.Insert(t, db, keyToDelete)

				r := testutil.SetupAuthenticatedRequest(http.MethodDelete, "/", nil, formattedKey)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
				ctx = context.WithValue(ctx, middleware.KeyContextKey, authKey)

				chiContext := chi.NewRouteContext()
				chiContext.URLParams.Add("id", authKey.ID)
				ctx = context.WithValue(ctx, chi.RouteCtxKey, chiContext)
				return r.WithContext(ctx), nil
			},
			validateFunc: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()

				var data errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				wantErr := errors.New(errors.BadRequest, "cannot delete key used for authentication")
				if diff := cmp.Diff(wantErr, &data, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
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

			w := httptest.NewRecorder()
			r, err := tc.setup(db, f, keygen)
			if err != nil {
				t.Fatalf("setup(): %v", err)
			}

			handler := http.HandlerFunc(deleteKey)
			handler.ServeHTTP(w, r)

			tc.validateFunc(t, w.Result())
		})
	}
}

func TestDeleteKey_ContextErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(f repository.Factory) (*http.Request, error)
		validateFunc func(t *testing.T, resp *http.Response)
	}{
		{
			name: "missingFactoryFromContext",
			setup: func(f repository.Factory) (*http.Request, error) {
				return httptest.NewRequest(http.MethodDelete, "/", nil), nil
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
			setup: func(f repository.Factory) (*http.Request, error) {
				r := httptest.NewRequest(http.MethodDelete, "/", nil)
				ctx := context.WithValue(r.Context(), middleware.FactoryContextKey, f)
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
			r, err := tc.setup(f)
			if err != nil {
				t.Fatalf("setup(): %v", err)
			}

			handler := http.HandlerFunc(deleteKey)
			handler.ServeHTTP(w, r)

			tc.validateFunc(t, w.Result())
		})
	}
}
