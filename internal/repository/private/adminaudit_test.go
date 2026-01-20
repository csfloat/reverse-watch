package private

import (
	"errors"
	"testing"
	"time"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"
	"reverse-watch/internal/util"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestAdminAuditRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	adminAuditRepo := NewAdminAuditRepository(db)

	testCases := []struct {
		name  string
		audit *models.AdminAudit
	}{
		{
			name: "addMarketplaceAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionAddMarketplace,
				TargetResourceType: models.TargetResourceTypeMarketplace,
				TargetResource:     "test-slug",
			},
		},
		{
			name: "updateMarketplaceAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionUpdateMarketplace,
				TargetResourceType: models.TargetResourceTypeMarketplace,
				TargetResource:     "test-slug",
				Details: testutil.MustRawJsonb(map[string]interface{}{
					"name":      "Test Marketplace",
					"is_active": false,
				}),
			},
		},
		{
			name: "removeMarketplaceAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionRemoveMarketplace,
				TargetResourceType: models.TargetResourceTypeMarketplace,
				TargetResource:     "test-slug",
			},
		},
		{
			name: "addKeyAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionAddKey,
				TargetResourceType: models.TargetResourceTypeKey,
				TargetResource:     "test-key-id",
				Details: testutil.MustRawJsonb(map[string]interface{}{
					"marketplace_slug": "test-slug",
					"permissions":      models.Permissions(8),
				}),
			},
		},
		{
			name: "removeKeyAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionRemoveKey,
				TargetResourceType: models.TargetResourceTypeKey,
				TargetResource:     "test-key-id",
				Details: testutil.MustRawJsonb(map[string]interface{}{
					"marketplace_slug": "test-slug",
					"permissions":      models.Permissions(8),
				}),
			},
		},
		{
			name: "updateReversalAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionUpdateReversal,
				TargetResourceType: models.TargetResourceTypeReversal,
				TargetResource:     "1",
				Details: testutil.MustRawJsonb(map[string]interface{}{
					"steam_id":         models.SteamID(76561197960265728),
					"marketplace_slug": "test-slug",
					"reversed_at":      time.Now().UnixMilli(),
					"expunged_at":      time.Now().UnixMilli(),
				}),
			},
		},
		{
			name: "removeReversalAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionRemoveReversal,
				TargetResourceType: models.TargetResourceTypeReversal,
				TargetResource:     "1",
			},
		},
		{
			name: "deleteUserDataAudit",
			audit: &models.AdminAudit{
				TargetAction:       models.TargetActionDeleteUserData,
				TargetResourceType: models.TargetResourceTypeReversal,
				Details: testutil.MustRawJsonb(map[string]interface{}{
					"steam_id":    models.SteamID(76561197960265728),
					"deleted_ids": []models.Snowflake{1},
				}),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := adminAuditRepo.Create(tc.audit); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			var storedAudit models.AdminAudit
			if err := db.Where("id = ?", tc.audit.ID).First(&storedAudit).Error; err != nil {
				t.Fatalf("failed to retrieve stored audit: %v", err)
			}

			if diff := cmp.Diff(*tc.audit, storedAudit, cmpopts.IgnoreFields(models.AdminAudit{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestAdminAuditRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	adminAuditRepo := NewAdminAuditRepository(db)

	testAudit := &models.AdminAudit{
		TargetAction:       models.TargetActionAddMarketplace,
		TargetResourceType: models.TargetResourceTypeMarketplace,
		TargetResource:     "test-slug",
	}
	testutil.Insert(t, db, testAudit)

	storedAudit, err := adminAuditRepo.Read(testAudit.ID)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if diff := cmp.Diff(testAudit, storedAudit, cmpopts.IgnoreFields(models.AdminAudit{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Error(diff)
	}
}

func TestAdminAuditRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	adminAuditRepo := NewAdminAuditRepository(db)

	testAudit := &models.AdminAudit{
		TargetAction:       models.TargetActionAddMarketplace,
		TargetResourceType: models.TargetResourceTypeMarketplace,
		TargetResource:     "test-slug",
	}
	testutil.Insert(t, db, testAudit)

	if err := adminAuditRepo.Delete(testAudit.ID); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var storedAudit models.AdminAudit
	err := db.Where("id = ?", testAudit.ID).First(&storedAudit).Error
	if err == nil {
		t.Fatalf("Delete(): got nil error, wanted %v", gorm.ErrRecordNotFound)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestAdminAuditRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	adminAuditRepo := NewAdminAuditRepository(db)

	err := adminAuditRepo.Delete(models.Snowflake(1))
	if err == nil {
		t.Fatalf("Delete(): got nil error, wanted %v", gorm.ErrRecordNotFound)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestAdminAuditRepository_List(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	adminAuditRepo := NewAdminAuditRepository(db)

	testAudits := []*models.AdminAudit{
		{
			TargetAction:       models.TargetActionAddMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-1",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"name":      "Test Marketplace 1",
				"is_active": true,
			}),
		},
		{
			TargetAction:       models.TargetActionAddMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-2",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"name":      "Test Marketplace 2",
				"is_active": true,
			}),
		},
		{
			TargetAction:       models.TargetActionUpdateMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-1",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"name":      "Test Marketplace 1",
				"is_active": false,
			}),
		},
		{
			TargetAction:       models.TargetActionUpdateMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-2",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"name":      "Test Marketplace 2",
				"is_active": false,
			}),
		},
		{
			TargetAction:       models.TargetActionRemoveMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-1",
		},
		{
			TargetAction:       models.TargetActionRemoveMarketplace,
			TargetResourceType: models.TargetResourceTypeMarketplace,
			TargetResource:     "test-slug-2",
		},
		{
			TargetAction:       models.TargetActionAddKey,
			TargetResourceType: models.TargetResourceTypeKey,
			TargetResource:     "test-key-id-1",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"marketplace_slug": "test-slug-1",
				"permissions":      models.Permissions(8),
			}),
		},
		{
			TargetAction:       models.TargetActionAddKey,
			TargetResourceType: models.TargetResourceTypeKey,
			TargetResource:     "test-key-id-2",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"marketplace_slug": "test-slug-2",
				"permissions":      models.Permissions(16),
			}),
		},
		{
			TargetAction:       models.TargetActionRemoveKey,
			TargetResourceType: models.TargetResourceTypeKey,
			TargetResource:     "test-key-id-1",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"marketplace_slug": "test-slug-1",
				"permissions":      models.Permissions(8),
			}),
		},
		{
			TargetAction:       models.TargetActionUpdateReversal,
			TargetResourceType: models.TargetResourceTypeReversal,
			TargetResource:     "1",
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"steam_id":         models.SteamID(76561197960265728),
				"marketplace_slug": "test-slug-1",
				"reversed_at":      time.Now().UnixMilli(),
				"expunged_at":      time.Now().UnixMilli(),
			}),
		},
		{
			TargetAction:       models.TargetActionRemoveReversal,
			TargetResourceType: models.TargetResourceTypeReversal,
			TargetResource:     "1",
		},
		{
			TargetAction:       models.TargetActionDeleteUserData,
			TargetResourceType: models.TargetResourceTypeReversal,
			Details: testutil.MustRawJsonb(map[string]interface{}{
				"steam_id":    models.SteamID(76561197960265728),
				"deleted_ids": []models.Snowflake{1},
			}),
		},
	}
	testutil.Insert(t, db, testAudits)

	testCases := []struct {
		name string
		opts *dto.AdminAuditListOptions
		want []*models.AdminAudit
	}{
		{
			name: "addMarketplace",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionAddMarketplace},
			},
			want: []*models.AdminAudit{testAudits[1], testAudits[0]},
		},
		{
			name: "updateMarketplace",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionUpdateMarketplace},
			},
			want: []*models.AdminAudit{testAudits[3], testAudits[2]},
		},
		{
			name: "removeMarketplace",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionRemoveMarketplace},
			},
			want: []*models.AdminAudit{testAudits[5], testAudits[4]},
		},
		{
			name: "addKey",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionAddKey},
			},
			want: []*models.AdminAudit{testAudits[7], testAudits[6]},
		},
		{
			name: "removeKey",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionRemoveKey},
			},
			want: []*models.AdminAudit{testAudits[8]},
		},
		{
			name: "updateReversal",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionUpdateReversal},
			},
			want: []*models.AdminAudit{testAudits[9]},
		},
		{
			name: "removeReversal",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionRemoveReversal},
			},
			want: []*models.AdminAudit{testAudits[10]},
		},
		{
			name: "deleteUserData",
			opts: &dto.AdminAuditListOptions{
				TargetActions: []models.TargetAction{models.TargetActionDeleteUserData},
			},
			want: []*models.AdminAudit{testAudits[11]},
		},
		{
			name: "allMarketplaceAudits",
			opts: &dto.AdminAuditListOptions{
				TargetResourceType: util.Ptr(models.TargetResourceTypeMarketplace),
			},
			want: []*models.AdminAudit{testAudits[5], testAudits[4], testAudits[3], testAudits[2], testAudits[1], testAudits[0]},
		},
		{
			name: "allKeyAudits",
			opts: &dto.AdminAuditListOptions{
				TargetResourceType: util.Ptr(models.TargetResourceTypeKey),
			},
			want: []*models.AdminAudit{testAudits[8], testAudits[7], testAudits[6]},
		},
		{
			name: "allReversalAudits",
			opts: &dto.AdminAuditListOptions{
				TargetResourceType: util.Ptr(models.TargetResourceTypeReversal),
			},
			want: []*models.AdminAudit{testAudits[11], testAudits[10], testAudits[9]},
		},
		{
			name: "targetResourceSlug",
			opts: &dto.AdminAuditListOptions{
				TargetResource: util.Ptr("test-slug-1"),
			},
			want: []*models.AdminAudit{testAudits[4], testAudits[2], testAudits[0]},
		},
		{
			name: "targetResourceKeyID",
			opts: &dto.AdminAuditListOptions{
				TargetResource: util.Ptr("test-key-id-1"),
			},
			want: []*models.AdminAudit{testAudits[8], testAudits[6]},
		},
		{
			name: "targetResourceReversalID",
			opts: &dto.AdminAuditListOptions{
				TargetResource: util.Ptr("1"),
			},
			want: []*models.AdminAudit{testAudits[10], testAudits[9]},
		},
		{
			name: "multipleFilters",
			opts: &dto.AdminAuditListOptions{
				TargetActions:  []models.TargetAction{models.TargetActionAddMarketplace},
				TargetResource: util.Ptr("test-slug-1"),
			},
			want: []*models.AdminAudit{testAudits[0]},
		},
		{
			name: "nilOptions",
			opts: nil,
			want: []*models.AdminAudit{testAudits[11], testAudits[10], testAudits[9], testAudits[8], testAudits[7], testAudits[6], testAudits[5], testAudits[4], testAudits[3], testAudits[2], testAudits[1], testAudits[0]},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotAudits, err := adminAuditRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}

			if diff := cmp.Diff(tc.want, gotAudits, cmpopts.IgnoreFields(models.AdminAudit{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}
