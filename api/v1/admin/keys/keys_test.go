package keys

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
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

func TestCreateKey(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error)
		validateFunc func(t *testing.T, db *gorm.DB, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "test-marketplace",
					Permissions:     models.PermissionWrite,
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
				var respData dto.RawKey
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.ID == "" {
					t.Error("expected key ID to be non-empty")
				}

				if respData.SecretKey == "" {
					t.Error("expected key to be non-empty")
				}

				if respData.MarketplaceSlug != "test-marketplace" {
					t.Errorf("wanted marketplace slug %q, got %q", "test-marketplace", respData.MarketplaceSlug)
				}

				if respData.Permissions != models.PermissionWrite {
					t.Errorf("wanted permissions %d, got %d", models.PermissionWrite, respData.Permissions)
				}

				var storedKey models.Key
				if err := db.Where("id = ?", respData.ID).First(&storedKey).Error; err != nil {
					t.Fatalf("failed to read key: %v", err)
				}

				if storedKey.MarketplaceSlug != "test-marketplace" {
					t.Errorf("wanted stored marketplace slug %q, got %q", "test-marketplace", storedKey.MarketplaceSlug)
				}

				if storedKey.Permissions != models.PermissionWrite {
					t.Errorf("wanted stored permissions %d, got %d", models.PermissionWrite, storedKey.Permissions)
				}

				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionAddKey, models.TargetResourceTypeKey, respData.ID).First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}
			},
		},
		{
			name: "validRequestWithManagePermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "test-marketplace",
					Permissions:     models.PermissionManage,
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
				var respData dto.RawKey
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}
				if respData.ID == "" {
					t.Error("expected key ID to be non-empty")
				}

				if respData.SecretKey == "" {
					t.Error("expected key to be non-empty")
				}

				if respData.MarketplaceSlug != "test-marketplace" {
					t.Errorf("wanted marketplace slug %q, got %q", "test-marketplace", respData.MarketplaceSlug)
				}

				if respData.Permissions != models.PermissionManage {
					t.Errorf("wanted permissions %d, got %d", models.PermissionManage, respData.Permissions)
				}

				var storedKey models.Key
				if err := db.Where("id = ?", respData.ID).First(&storedKey).Error; err != nil {
					t.Fatalf("failed to read key: %v", err)
				}

				if storedKey.Permissions != models.PermissionManage {
					t.Errorf("wanted stored permissions %d, got %d", models.PermissionManage, storedKey.Permissions)
				}

				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionAddKey, models.TargetResourceTypeKey, respData.ID).First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}
			},
		},
		{
			name: "invalidRequestBody",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
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
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "",
					Permissions:     models.PermissionWrite,
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

				if diff := cmp.Diff(errors.BadRequest, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}

				if respData.Details != "marketplace slug and permissions are required" {
					t.Errorf("wanted details %q, got %q", "marketplace slug and permissions are required", respData.Details)
				}
			},
		},
		{
			name: "zeroPermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "test-marketplace",
					Permissions:     0,
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

				if diff := cmp.Diff(errors.BadRequest, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}

				if respData.Details != "marketplace slug and permissions are required" {
					t.Errorf("wanted details %q, got %q", "marketplace slug and permissions are required", respData.Details)
				}
			},
		},
		{
			name: "adminKeyForNonCSFloatMarketplace",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "test-marketplace",
					Permissions:     models.PermissionAdmin,
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

				if diff := cmp.Diff(errors.BadRequest, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}

				if respData.Details != "admin scoped keys can only be created for csfloat" {
					t.Errorf("wanted details %q, got %q", "admin scoped keys can only be created for csfloat", respData.Details)
				}
			},
		},
		{
			name: "emptyMarketplaceSlugAndZeroPermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "",
					Permissions:     0,
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

				if diff := cmp.Diff(errors.BadRequest, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}

				if respData.Details != "marketplace slug and permissions are required" {
					t.Errorf("wanted details %q, got %q", "marketplace slug and permissions are required", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "test-marketplace",
					Permissions:     models.PermissionWrite,
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
		{
			name: "nonExistentMarketplace",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reqBody := struct {
					MarketplaceSlug string             `json:"marketplace_slug"`
					Permissions     models.Permissions `json:"permissions"`
				}{
					MarketplaceSlug: "nonexistent-marketplace",
					Permissions:     models.PermissionWrite,
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
					t.Errorf("wanted status code %d, got %d", http.StatusNotFound, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(errors.InvalidReference, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
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
			permissionsMiddleware := middleware.RequirePermissions(models.PermissionAdmin)
			handler := http.HandlerFunc(createKey)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, err := tc.setup(t, db, keygen)
			if err != nil {
				t.Fatal(err)
			}

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, w.Result())
		})
	}
}
