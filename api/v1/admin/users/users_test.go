package users

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestPurgeUser(t *testing.T) {
	t.Parallel()
	logging.Initialize()

	testCases := []struct {
		name         string
		setup        func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error)
		validateFunc func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response)
	}{
		{
			name: "validRequest",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				testMarketplace, _, _ := testutil.SetupMarketplaceWithKey(t, db, "test-marketplace", keygen, models.PermissionWrite)

				steamID := models.SteamID(76561197960287930)
				reversals := []*models.Reversal{
					{
						Model:           models.Model{ID: 1},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplace.Slug,
						Source:          util.Ptr(models.SourceDirect),
						ReversedAt:      1717756800,
					},
					{
						Model:           models.Model{ID: 2},
						SteamID:         steamID,
						MarketplaceSlug: testMarketplace.Slug,
						Source:          util.Ptr(models.SourceUserReport),
						ReversedAt:      1717756900,
					},
					{
						Model:           models.Model{ID: 3},
						SteamID:         models.SteamID(76561197960287931),
						MarketplaceSlug: testMarketplace.Slug,
						ReversedAt:      1717757000,
					},
				}
				testutil.Insert(t, db, reversals...)

				r := httptest.NewRequest(http.MethodDelete, "/"+steamID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, &steamID, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("wanted status code %d, got %d", http.StatusOK, resp.StatusCode)
				}

				var reversals []models.Reversal
				err := db.Unscoped().Where("steam_id = ?", steamID).Find(&reversals).Error
				if err != nil {
					t.Fatalf("failed to query reversals: %v", err)
				}

				if len(reversals) > 0 {
					t.Errorf("wanted all reversals to be deleted, found %d", len(reversals))
				}

				// Ensure other reversals still exist
				err = db.Unscoped().Find(&reversals).Error
				if err != nil {
					t.Fatalf("failed to query reversals: %v", err)
				}

				wantReversal := &models.Reversal{
					Model:           models.Model{ID: 3},
					SteamID:         models.SteamID(76561197960287931),
					MarketplaceSlug: "test-marketplace",
					ReversedAt:      1717757000,
				}

				if diff := cmp.Diff(wantReversal, &reversals[0], cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt")); diff != "" {
					t.Error(diff)
				}

				var audit models.AdminAudit
				if err := db.Model(&models.AdminAudit{}).Where("target_action = ? AND target_resource_type = ? AND target_resource = ?",
					models.TargetActionDeleteUserData, models.TargetResourceTypeUser, steamID.String()).First(&audit).Error; err != nil {
					t.Fatalf("failed to find admin audit: %v", err)
				}

				var gotDetails struct {
					Key string `json:"key"`
				}
				if err := json.Unmarshal(audit.Details.Raw, &gotDetails); err != nil {
					t.Fatalf("failed to unmarshal audit details: %v", err)
				}

				if gotDetails.Key != key {
					t.Errorf("wanted key %q, got %q", key, gotDetails.Key)
				}
			},
		},
		{
			name: "emptySteamID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				r := httptest.NewRequest(http.MethodDelete, "/", nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, nil, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response) {
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
			},
		},
		{
			name: "invalidSteamID",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				steamId := models.SteamID(712)
				r := httptest.NewRequest(http.MethodDelete, "/"+steamId.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, &steamId, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response) {
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

				if respData.Details != "invalid steam id: 712" {
					t.Errorf("wanted details %q, got %q", "invalid steam id: 712", respData.Details)
				}
			},
		},
		{
			name: "userNotFound",
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error) {
				_, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionAdmin)

				steamID := models.SteamID(76561197960287930)
				r := httptest.NewRequest(http.MethodDelete, "/"+steamID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, &steamID, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response) {
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
			setup: func(t *testing.T, db *gorm.DB, keygen isecret.KeyGenerator) (*http.Request, *models.SteamID, string, error) {
				testMarketplace, authKey, formattedKey := testutil.SetupMarketplaceWithKey(t, db, "csfloat", keygen, models.PermissionWrite)

				steamID := models.SteamID(76561197960287930)
				reversal := &models.Reversal{
					Model:           models.Model{ID: 1},
					SteamID:         steamID,
					MarketplaceSlug: testMarketplace.Slug,
					Source:          util.Ptr(models.SourceDirect),
					ReversedAt:      1717756800,
				}
				testutil.Insert(t, db, reversal)

				r := httptest.NewRequest(http.MethodDelete, "/"+steamID.String(), nil)
				r.Header.Set("Authorization", "Bearer "+formattedKey)

				return r, &steamID, authKey.ID, nil
			},
			validateFunc: func(t *testing.T, db *gorm.DB, steamID *models.SteamID, key string, resp *http.Response) {
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
				if err := db.Where("steam_id = ?", steamID).First(&storedReversal).Error; err != nil {
					t.Fatalf("failed to read reversal: %v", err)
				}

				if storedReversal.SteamID != *steamID {
					t.Error("expected reversal to not be deleted due to insufficient permissions")
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
			handler := http.HandlerFunc(purgeUser)

			finalHandler := factoryMiddleware(
				middleware.AuthMiddleware(
					permissionsMiddleware(handler),
				),
			)

			w := httptest.NewRecorder()
			r, steamID, key, err := tc.setup(t, db, keygen)
			if err != nil {
				t.Fatal(err)
			}

			chiContext := chi.NewRouteContext()
			if steamID != nil {
				chiContext.URLParams.Add("steamId", steamID.String())
			}
			ctx := context.WithValue(r.Context(), chi.RouteCtxKey, chiContext)
			r = r.WithContext(ctx)

			finalHandler.ServeHTTP(w, r)

			tc.validateFunc(t, db, steamID, key, w.Result())
		})
	}
}
