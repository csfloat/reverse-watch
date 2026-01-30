package keys

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
		setup          func(db *gorm.DB, factory repository.Factory) (*http.Request, error)
		wantStatusCode int
		wantRawKey     bool
	}{
		{
			name: "validRequest",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				testKey := &models.Key{
					ID:              "test-key",
					MarketplaceSlug: "test-marketplace",
					Environment:     constants.EnvironmentDevelopment,
					Permissions:     models.PermissionWrite,
				}

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
				ctx := context.WithValue(r.Context(), middleware.KeyContextKey, testKey)
				ctx = context.WithValue(ctx, middleware.FactoryContextKey, factory)
				return r.WithContext(ctx), nil
			},
			wantStatusCode: http.StatusOK,
			wantRawKey:     true,
		},
		{
			name: "noKeyInContext",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				return http.NewRequest(http.MethodPost, "/", nil)
			},
			wantStatusCode: http.StatusInternalServerError,
			wantRawKey:     false,
		},
		{
			name: "invalidBody",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
				testKey := &models.Key{
					ID:              "test-key",
					MarketplaceSlug: "test-marketplace",
					Environment:     constants.EnvironmentDevelopment,
					Permissions:     models.PermissionWrite,
				}

				body := `{"permissions": 1}`
				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte(body)))
				ctx := context.WithValue(r.Context(), middleware.KeyContextKey, testKey)
				ctx = context.WithValue(ctx, middleware.FactoryContextKey, factory)
				return r.WithContext(ctx), nil
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalidPermissions",
			setup: func(db *gorm.DB, factory repository.Factory) (*http.Request, error) {
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
			wantStatusCode: http.StatusBadRequest,
			wantRawKey:     false,
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
			wantRawKey:     false,
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
