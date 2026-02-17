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
				if resp.StatusCode != http.StatusConflict {
					t.Errorf("wanted status code %d, got %d", http.StatusConflict, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData errors.Error
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Details != "UNIQUE constraint failed: marketplaces.slug" {
					t.Errorf("wanted details %q, got %q", "UNIQUE constraint failed: marketplaces.slug", respData.Details)
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

func TestUpdateMarketplace(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error)
		validateFunc func(t *testing.T, db *gorm.DB, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				// Setup CSFloat marketplace with admin key
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				// Create marketplace to update
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				newName := "Updated Marketplace Name"
				isActive := false
				reqBody := dto.MarketplaceUpdates{
					Name:     &newName,
					IsActive: &isActive,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/test-marketplace", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Marketplace
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Slug != "test-marketplace" {
					t.Errorf("wanted marketplace slug %q, got %q", "test-marketplace", respData.Slug)
				}

				if respData.Name != "Updated Marketplace Name" {
					t.Errorf("wanted marketplace name %q, got %q", "Updated Marketplace Name", respData.Name)
				}

				if respData.IsActive {
					t.Error("expected marketplace to be inactive")
				}

				// Verify marketplace was updated in database
				var storedMarketplace models.Marketplace
				if err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				if diff := cmp.Diff(&respData, &storedMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
					t.Error(diff)
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionUpdateMarketplace, models.TargetResourceTypeMarketplace, "test-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				wantName := "Updated Marketplace Name"
				wantActive := false
				wantDetails, err := models.ToRawJsonb(dto.MarketplaceUpdates{
					Name:     &wantName,
					IsActive: &wantActive,
				})
				if err != nil {
					t.Fatalf("ToRawJsonb(): %v", err)
				}

				gotDetails := models.RawJsonb{
					Raw: make(json.RawMessage, 0),
				}
				if err := json.Unmarshal(audit.Details.Raw, &gotDetails); err != nil {
					t.Fatalf("Unmarshal(): %v", err)
				}

				if diff := cmp.Diff(wantDetails, &gotDetails); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "updateNameOnly",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				newName := "New Name Only"
				reqBody := dto.MarketplaceUpdates{
					Name: &newName,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/test-marketplace", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Marketplace
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Name != "New Name Only" {
					t.Errorf("wanted marketplace name %q, got %q", "New Name Only", respData.Name)
				}

				if !respData.IsActive {
					t.Error("expected marketplace to still be active")
				}

				// Verify marketplace was updated in database
				var storedMarketplace models.Marketplace
				if err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				if diff := cmp.Diff(&respData, &storedMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
					t.Error(diff)
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionUpdateMarketplace, models.TargetResourceTypeMarketplace, "test-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				wantName := "New Name Only"
				wantDetails, err := models.ToRawJsonb(dto.MarketplaceUpdates{
					Name: &wantName,
				})
				if err != nil {
					t.Fatalf("ToRawJsonb(): %v", err)
				}

				gotDetails := models.RawJsonb{
					Raw: make(json.RawMessage, 0),
				}
				if err := json.Unmarshal(audit.Details.Raw, &gotDetails); err != nil {
					t.Fatalf("Unmarshal(): %v", err)
				}

				if diff := cmp.Diff(wantDetails, &gotDetails); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "updateIsActiveOnly",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				isActive := false
				reqBody := dto.MarketplaceUpdates{
					IsActive: &isActive,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/test-marketplace", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Marketplace
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.IsActive {
					t.Error("expected marketplace to be inactive")
				}

				if respData.Name != "Test Marketplace" {
					t.Errorf("wanted marketplace name %q, got %q", "Test Marketplace", respData.Name)
				}

				// Verify marketplace was updated in database
				var storedMarketplace models.Marketplace
				if err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				if diff := cmp.Diff(&respData, &storedMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
					t.Error(diff)
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionUpdateMarketplace, models.TargetResourceTypeMarketplace, "test-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				wantActive := false
				wantDetails, err := models.ToRawJsonb(dto.MarketplaceUpdates{
					IsActive: &wantActive,
				})
				if err != nil {
					t.Fatalf("ToRawJsonb(): %v", err)
				}

				gotDetails := models.RawJsonb{
					Raw: make(json.RawMessage, 0),
				}
				if err := json.Unmarshal(audit.Details.Raw, &gotDetails); err != nil {
					t.Fatalf("Unmarshal(): %v", err)
				}

				if diff := cmp.Diff(wantDetails, &gotDetails); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "emptySlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				newName := "Updated Name"
				reqBody := dto.MarketplaceUpdates{
					Name: &newName,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "", nil
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

				if respData.Details != "slug cannot be empty" {
					t.Errorf("wanted details %q, got %q", "slug cannot be empty", respData.Details)
				}
			},
		},
		{
			name: "invalidRequestBody",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodPatch, "/test-marketplace", bytes.NewBuffer([]byte("invalid json")))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
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

				if diff := cmp.Diff(errors.JSONDecode, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}
			},
		},
		{
			name: "marketplaceNotFound",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				newName := "Updated Name"
				reqBody := dto.MarketplaceUpdates{
					Name: &newName,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/nonexistent-marketplace", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "nonexistent-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
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

				if respData.Details != "record not found" {
					t.Errorf("wanted details %q, got %q", "record not found", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				newName := "Updated Name"
				reqBody := dto.MarketplaceUpdates{
					Name: &newName,
				}

				payload, err := json.Marshal(reqBody)
				if err != nil {
					return nil, "", err
				}

				r := httptest.NewRequest(http.MethodPatch, "/test-marketplace", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
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

				if diff := cmp.Diff(errors.Forbidden, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
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
			handler := http.HandlerFunc(updateMarketplace)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, slug, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			// Add URL parameter using chi's context
			chiContext := chi.NewRouteContext()
			chiContext.URLParams.Add("slug", slug)
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
			r = r.WithContext(ctx)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, w.Result())
		})
	}
}

func TestDeleteMarketplace(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error)
		validateFunc func(t *testing.T, db *gorm.DB, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				// Setup CSFloat marketplace with admin key
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				// Create marketplace to delete
				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				r := httptest.NewRequest(http.MethodDelete, "/test-marketplace", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				// Verify marketplace was deleted from database
				var storedMarketplace models.Marketplace
				err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error
				if err == nil {
					t.Error("expected marketplace to be deleted, but it still exists")
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionRemoveMarketplace, models.TargetResourceTypeMarketplace, "test-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
				}

				if audit.Details != nil {
					t.Errorf("expected audit details to be nil, got %v", audit.Details)
				}
			},
		},
		{
			name: "emptySlug",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "", nil
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

				if respData.Details != "slug cannot be empty" {
					t.Errorf("wanted details %q, got %q", "slug cannot be empty", respData.Details)
				}
			},
		},
		{
			name: "marketplaceNotFound",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/nonexistent-marketplace", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "nonexistent-marketplace", nil
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

				if diff := cmp.Diff(errors.DBDelete, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped", "Details")); diff != "" {
					t.Error(diff)
				}

				if respData.Details != "failed to delete marketplace" {
					t.Errorf("wanted details %q, got %q", "failed to delete marketplace", respData.Details)
				}
			},
		},
		{
			name: "invalidPermissions",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)

				testMarketplace := &models.Marketplace{
					Slug:     "test-marketplace",
					Name:     "Test Marketplace",
					IsActive: true,
				}
				testutil.Insert(t, db, testMarketplace)

				r := httptest.NewRequest(http.MethodDelete, "/test-marketplace", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "test-marketplace", nil
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

				if diff := cmp.Diff(errors.Forbidden, respData, cmpopts.IgnoreFields(errors.Error{}, "status", "wrapped")); diff != "" {
					t.Error(diff)
				}

				// Verify marketplace was NOT deleted
				var storedMarketplace models.Marketplace
				if err := db.Where("slug = ?", "test-marketplace").First(&storedMarketplace).Error; err != nil {
					t.Fatalf("expected marketplace to still exist: %v", err)
				}
			},
		},
		{
			name: "deleteInactiveMarketplace",
			setup: func(t *testing.T, db *gorm.DB, f repository.Factory, keygen isecret.KeyGenerator) (*http.Request, string, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				// Create inactive marketplace to delete
				testMarketplace := &models.Marketplace{
					Slug:     "inactive-marketplace",
					Name:     "Inactive Marketplace",
					IsActive: false,
				}
				testutil.Insert(t, db, testMarketplace)

				r := httptest.NewRequest(http.MethodDelete, "/inactive-marketplace", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, "inactive-marketplace", nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				// Verify marketplace was deleted from database
				var storedMarketplace models.Marketplace
				err := db.Where("slug = ?", "inactive-marketplace").First(&storedMarketplace).Error
				if err == nil {
					t.Error("expected marketplace to be deleted, but it still exists")
				}

				// Verify admin audit was created
				var audit models.AdminAudit
				if err := db.Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionRemoveMarketplace, models.TargetResourceTypeMarketplace, "inactive-marketplace").First(&audit).Error; err != nil {
					t.Fatalf("First(): %v", err)
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
			handler := http.HandlerFunc(deleteMarketplace)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, slug, err := tc.setup(t, db, f, keygen)
			if err != nil {
				t.Fatal(err)
			}

			// Add URL parameter using chi's context
			chiContext := chi.NewRouteContext()
			chiContext.URLParams.Add("slug", slug)
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
			r = r.WithContext(ctx)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, w.Result())
		})
	}
}
