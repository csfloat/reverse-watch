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

func TestDeleteKey(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error)
		validateFunc func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)
				_, keyToDelete, _ := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionManage)

				r := httptest.NewRequest(http.MethodDelete, "/"+keyToDelete.ID, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, keyToDelete.ID, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				var deletedKey models.Key
				if err := db.Unscoped().Where("id = ?", keyID).First(&deletedKey).Error; err != nil {
					t.Fatalf("failed to query deleted key: %v", err)
				}

				if deletedKey.DeletedAt.Time.IsZero() {
					t.Error("expected key to be soft deleted")
				}

				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionRemoveKey, models.TargetResourceTypeKey, keyID).First(&audit).Error; err != nil {
					t.Fatalf("failed to find admin audit: %v", err)
				}

				wantDetails := &dto.KeyAuditDetails{
					MarketplaceSlug: "test-marketplace",
					Permissions:     models.PermissionManage,
					AdminKey:        authKeyID,
				}

				var details *dto.KeyAuditDetails
				if err := json.Unmarshal(audit.Details.Raw, &details); err != nil {
					t.Fatalf("failed to unmarshal audit details: %v", err)
				}

				if diff := cmp.Diff(wantDetails, details); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "emptyID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "", authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response) {
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

				if respData.Details != "id must not be empty" {
					t.Errorf("wanted details %q, got %q", "id must not be empty", respData.Details)
				}
			},
		},
		{
			name: "selfDeleteKey",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/"+authKey.ID, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)
				
				return r, authKey.ID, formattedKey, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response) {
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

				if respData.Details != "cannot delete key used for authentication" {
					t.Errorf("wanted details %q, got %q", "cannot delete key used for authentication", respData.Details)
				}
			},
		},
		{
			name: "nonExistentKey",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/nonexistent-key-id", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "nonexistent-key-id", authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response) {
				if resp.StatusCode != http.StatusNotFound {
					t.Errorf("wanted status code %d, got %d", http.StatusNotFound, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if diff := cmp.Diff(errors.NotFound, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "insufficientPermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, string, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)
				_, keyToDelete, _ := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionManage)

				r := httptest.NewRequest(http.MethodDelete, "/"+keyToDelete.ID, nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, keyToDelete.ID, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, keyID, authKeyID string, resp *http.Response) {
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

				var existingKey models.Key
				if err := db.Where("id = ?", keyID).First(&existingKey).Error; err != nil {
					t.Fatalf("failed to query key: %v", err)
				}

				if !existingKey.DeletedAt.Time.IsZero() {
					t.Error("expected key to not be deleted due to insufficient permissions")
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
			handler := http.HandlerFunc(deleteKey)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, keyID, authKeyID, err := tc.setup(t, db, keygen)
			if err != nil {
				t.Fatal(err)
			}

			chiContext := chi.NewRouteContext()
			chiContext.URLParams.Add("id", keyID)
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
			r = r.WithContext(ctx)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, keyID, authKeyID, w.Result())
		})
	}
}
