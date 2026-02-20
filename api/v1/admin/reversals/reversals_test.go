package reversals

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
	"reverse-watch/util"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestModifyReversal(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error)
		validateFunc func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response)
	}{
		{
			name: "validUpdateSource",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					Source:          util.Ptr(models.SourceDirect),
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				updates := dto.ReversalUpdates{
					Source: util.Ptr(models.SourceUserReport),
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Reversal
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Source == nil || *respData.Source != models.SourceUserReport {
					t.Errorf("wanted source %d, got %v", models.SourceUserReport, respData.Source)
				}

				if respData.ID != originalReversal.ID {
					t.Errorf("wanted ID %d, got %d", originalReversal.ID, respData.ID)
				}

				var storedReversal models.Reversal
				if err := db.Where("id = ?", originalReversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.Source == nil || *storedReversal.Source != models.SourceUserReport {
					t.Errorf("wanted stored source %d, got %v", models.SourceUserReport, storedReversal.Source)
				}

				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionUpdateReversal, models.TargetResourceTypeReversal, originalReversal.ID.String()).First(&audit).Error; err != nil {
					t.Fatalf("failed to find admin audit: %v", err)
				}
			},
		},
		{
			name: "validUpdateRelatedSteamID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					Source:          util.Ptr(models.SourceRelatedUser),
					RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				newRelatedSteamID := models.SteamID(76561197960287932)
				updates := dto.ReversalUpdates{
					Source:         util.Ptr(models.SourceRelatedUser),
					RelatedSteamID: &newRelatedSteamID,
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Reversal
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				expectedSteamID := models.SteamID(76561197960287932)
				if respData.RelatedSteamID == nil || *respData.RelatedSteamID != expectedSteamID {
					t.Errorf("wanted related steam id %d, got %v", expectedSteamID, respData.RelatedSteamID)
				}

				var storedReversal models.Reversal
				if err := db.Where("id = ?", originalReversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.RelatedSteamID == nil || *storedReversal.RelatedSteamID != expectedSteamID {
					t.Errorf("wanted stored related steam id %d, got %v", expectedSteamID, storedReversal.RelatedSteamID)
				}
			},
		},
		{
			name: "validUpdateMultipleFields",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					Source:          util.Ptr(models.SourceDirect),
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				newReversedAt := uint64(1735689600100)
				updates := dto.ReversalUpdates{
					Source:     util.Ptr(models.SourceUserReport),
					ReversedAt: &newReversedAt,
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				defer resp.Body.Close()
				var respData models.Reversal
				if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
					t.Fatalf("failed to decode response body: %v", err)
				}

				if respData.Source == nil || *respData.Source != models.SourceUserReport {
					t.Errorf("wanted source %d, got %v", models.SourceUserReport, respData.Source)
				}

				if respData.ReversedAt != 1735689600100 {
					t.Errorf("wanted reversed at %d, got %d", 1735689600100, respData.ReversedAt)
				}

				var storedReversal models.Reversal
				if err := db.Where("id = ?", originalReversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.Source == nil || *storedReversal.Source != models.SourceUserReport {
					t.Errorf("wanted stored source %d, got %v", models.SourceUserReport, storedReversal.Source)
				}

				if storedReversal.ReversedAt != 1735689600100 {
					t.Errorf("wanted stored reversed at %d, got %d", 1735689600100, storedReversal.ReversedAt)
				}
			},
		},
		{
			name: "emptyID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				updates := dto.ReversalUpdates{
					Source: util.Ptr(models.SourceUserReport),
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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
			name: "invalidID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				updates := dto.ReversalUpdates{
					Source: util.Ptr(models.SourceUserReport),
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/invalid-id", bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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

				if respData.Details != "invalid id" {
					t.Errorf("wanted details %q, got %q", "invalid id", respData.Details)
				}
			},
		},
		{
			name: "invalidRequestBody",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer([]byte("invalid json")))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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
			name: "nonExistentReversal",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				_, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				updates := dto.ReversalUpdates{
					Source: util.Ptr(models.SourceUserReport),
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				nonExistentID := models.Snowflake(999999)
				r := httptest.NewRequest(http.MethodPatch, "/"+nonExistentID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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
			name: "invalidSourceAndRelatedSteamIDCombination",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				relatedSteamID := models.SteamID(76561197960287931)
				updates := dto.ReversalUpdates{
					Source:         util.Ptr(models.SourceDirect),
					RelatedSteamID: &relatedSteamID,
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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

				if respData.Details != "invalid related_steam_id and source combination" {
					t.Errorf("wanted details %q, got %q", "invalid related_steam_id and source combination", respData.Details)
				}
			},
		},
		{
			name: "insufficientPermissions",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.Reversal, error) {
				testMarketplace, _, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)

				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: testMarketplace.Slug,
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				updates := dto.ReversalUpdates{
					Source: util.Ptr(models.SourceUserReport),
				}

				payload, err := json.Marshal(updates)
				if err != nil {
					return nil, nil, err
				}

				r := httptest.NewRequest(http.MethodPatch, "/"+reversal.ID.String(), bytes.NewBuffer(payload))
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, reversal, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, originalReversal *models.Reversal, resp *http.Response) {
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

				var storedReversal models.Reversal
				if err := db.Where("id = ?", originalReversal.ID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.Source != originalReversal.Source {
					t.Error("expected reversal to not be modified due to insufficient permissions")
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
			handler := http.HandlerFunc(modifyReversal)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, reversal, err := tc.setup(t, db, keygen)
			if err != nil {
				t.Fatal(err)
			}

			chiContext := chi.NewRouteContext()
			if reversal != nil {
				chiContext.URLParams.Add("id", reversal.ID.String())
			} else {
				urlID := chi.URLParam(r, "id")
				if urlID == "" {
					if r.URL.Path != "/" {
						urlID = r.URL.Path[1:]
					}
				}
				chiContext.URLParams.Add("id", urlID)
			}
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
			r = r.WithContext(ctx)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, reversal, w.Result())
		})
	}
}
