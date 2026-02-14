package marketplace

import (
	"bytes"
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

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestOnboardMarketplace(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error)
		validateFunc func(t *testing.T, db *gorm.DB, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				// Setup CSFloat marketplace with admin key
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "test-marketplace",
					Name:            "Test Marketplace",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData struct {
					Marketplace *models.Marketplace `json:"marketplace"`
					Key         *dto.RawKey         `json:"key"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Marketplace == nil {
					t.Fatal("expected marketplace to be non-nil")
				}

				if respData.Key == nil {
					t.Fatal("expected key to be non-nil")
				}

				if respData.Marketplace.Slug != "test-marketplace" {
					t.Errorf("wanted marketplace slug %q, got %q", "test-marketplace", respData.Marketplace.Slug)
				}

				if respData.Marketplace.Name != "Test Marketplace" {
					t.Errorf("wanted marketplace name %q, got %q", "Test Marketplace", respData.Marketplace.Name)
				}

				if !respData.Marketplace.IsActive {
					t.Error("expected marketplace to be active")
				}

				// Verify marketplace was created in database
				var storedMarketplace models.Marketplace
				if err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				if diff := cmp.Diff(respData.Marketplace, &storedMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
					t.Error(diff)
				}

				// Verify key was created
				var storedKey models.Key
				if err := db.Where("id = ?", respData.Key.ID).First(&storedKey).Error; err != nil {
					t.Fatalf("failed to read key: %v", err)
				}

				if storedKey.Permissions != models.PermissionManage {
					t.Errorf("wanted key permissions %d, got %d", models.PermissionManage, storedKey.Permissions)
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionAddMarketplace, models.TargetResourceTypeMarketplace, "test-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}
			},
		},
		{
			name: "invalidRequestBody",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("invalid json")))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Message != "failed to decode JSON" {
					t.Errorf("wanted message %q, got %q", "failed to decode JSON", respData.Message)
				}
			},
		},
		{
			name: "emptyMarketplaceSlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "",
					Name:            "Test Marketplace",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "fields cannot be empty" {
					t.Errorf("wanted details %q, got %q", "fields cannot be empty", respData.Details)
				}
			},
		},
		{
			name: "emptyName",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "test-marketplace",
					Name:            "",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "fields cannot be empty" {
					t.Errorf("wanted details %q, got %q", "fields cannot be empty", respData.Details)
				}
			},
		},
		{
			name: "duplicateMarketplaceSlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				// Pre-create marketplace with same slug
				existingMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Existing Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, existingMarketplace)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "test-marketplace",
					Name:            "Test Marketplace",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusInternalServerError {
					t.Errorf("wanted status code %d, got %d", http.StatusInternalServerError, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "failed to create marketplace: UNIQUE constraint failed: marketplaces.slug" {
					t.Errorf("wanted details %q, got %q", "failed to create marketplace: UNIQUE constraint failed: marketplaces.slug", respData.Details)
				}
			},
		},
		{
			name: "invalidSlugInvalidName",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "",
					Name:            "",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusBadRequest {
					t.Errorf("wanted status code %d, got %d", http.StatusBadRequest, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "fields cannot be empty" {
					t.Errorf("wanted details %q, got %q", "fields cannot be empty", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				reqBody := onboardMarketplaceRequest{
					MarketplaceSlug: "new-marketplace",
					Name:            "New Marketplace",
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, err
				}

				r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusForbidden {
					t.Errorf("wanted status code %d, got %d", http.StatusForbidden, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Message != "forbidden" {
					t.Errorf("wanted message %q, got %q", "forbidden", respData.Message)
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionAdmin)
			handler := http.HandlerFunc(onboardMarketplace)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, w.Result())
		})
	}
}
