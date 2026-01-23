package private

import (
	"errors"
	"sort"
	"testing"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/models/constants"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestKeyRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	csfloatTestMarketplace := &models.Marketplace{
		Slug: "csfloat",
		Name: "CSFloat",
	}
	testutil.Insert(t, db, testMarketplace, csfloatTestMarketplace)

	testCases := []struct {
		name string
		key  *models.Key
	}{
		{
			name: "validKey",
			key: &models.Key{
				ID:              "test-key-id",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionWrite,
			},
		},
		{
			name: "adminKeyForCSFloat",
			key: &models.Key{
				ID:              "admin-key-id",
				Environment:     constants.EnvironmentDevelopment,
				MarketplaceSlug: csfloatTestMarketplace.Slug,
				Permissions:     models.PermissionAdmin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := keyRepo.Create(tc.key); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			if tc.key.CreatedAt == 0 {
				t.Errorf("got CreatedAt %d, wanted non-zero value", tc.key.CreatedAt)
			}
			if tc.key.UpdatedAt == 0 {
				t.Errorf("got UpdatedAt %d, wanted non-zero value", tc.key.UpdatedAt)
			}
		})
	}
}

func TestKeyRepository_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	csfloatTestMarketplace := &models.Marketplace{
		Slug: "csfloat",
		Name: "CSFloat",
	}
	testutil.Insert(t, db, testMarketplace, csfloatTestMarketplace)

	testCases := []struct {
		name    string
		key     *models.Key
		wantErr string
	}{
		{
			name: "emptyID",
			key: &models.Key{
				ID:              "",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionRead,
			},
			wantErr: "id is required",
		},
		{
			name: "emptyEnvironment",
			key: &models.Key{
				ID:              "test-key-id-2",
				Environment:     "",
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionExport | models.PermissionRead,
			},
			wantErr: "environment is required",
		},
		{
			name: "emptyMarketplaceSlug",
			key: &models.Key{
				ID:              "test-key-id-3",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "",
				Permissions:     models.PermissionExport,
			},
			wantErr: "marketplace_slug is required",
		},
		{
			name: "noPermissions",
			key: &models.Key{
				ID:              "test-key-id-4",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionNone,
			},
			wantErr: "at least one permission is required",
		},
		{
			name: "adminPermissionNonCSFloat",
			key: &models.Key{
				ID:              "test-key-id-5",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "test-marketplace",
				Permissions:     models.PermissionAdmin,
			},
			wantErr: "admin scoped keys can only be created for CSFloat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := keyRepo.Create(tc.key)
			if err == nil {
				t.Fatal("Create(): got nil error, wanted error from BeforeCreate")
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Create(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}

func TestKeyRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testCSFloatMarketplace := &models.Marketplace{
		Slug: "csfloat",
		Name: "CSFloat",
	}
	testutil.Insert(t, db, testMarketplace, testCSFloatMarketplace)

	testCases := []struct {
		name string
		key  *models.Key
	}{
		{
			name: "singlePermission",
			key: &models.Key{
				ID:              "test-key-id-1",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionRead,
			},
		},
		{
			name: "multiplePermissions",
			key: &models.Key{
				ID:              "test-key-id-2",
				Environment:     constants.EnvironmentDevelopment,
				MarketplaceSlug: testMarketplace.Slug,
				Permissions:     models.PermissionWrite | models.PermissionManage | models.PermissionExport,
			},
		},
		{
			name: "adminPermissionCSFloat",
			key: &models.Key{
				ID:              "test-key-id-3",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: testCSFloatMarketplace.Slug,
				Permissions:     models.PermissionAdmin,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := keyRepo.Create(tc.key); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			var storedKey models.Key
			if err := db.Where("id = ?", tc.key.ID).First(&storedKey).Error; err != nil {
				t.Fatalf("failed to retrieve stored key: %v", err)
			}

			if diff := cmp.Diff(*tc.key, storedKey, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestKeyRepository_Create_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testCSFloatMarketplace := &models.Marketplace{
		Slug: "csfloat",
		Name: "CSFloat",
	}
	testutil.Insert(t, db, testMarketplace, testCSFloatMarketplace)

	testCases := []struct {
		name    string
		key     *models.Key
		wantErr string
	}{
		{
			name: "nonExistentMarketplace",
			key: &models.Key{
				ID:              "test-key-id",
				Environment:     constants.EnvironmentProduction,
				MarketplaceSlug: "non-existent-marketplace",
				Permissions:     models.PermissionRead,
			},
			wantErr: "FOREIGN KEY constraint failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := keyRepo.Create(tc.key)
			if err == nil {
				t.Fatal("Create(): got nil error, wanted error")
			}

			if err.Error() != tc.wantErr {
				t.Errorf("Create(): got error %q, wanted %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestKeyRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testutil.Insert(t, db, testMarketplace)

	testKey := &models.Key{
		ID:              "test-key-id",
		Environment:     constants.EnvironmentProduction,
		MarketplaceSlug: testMarketplace.Slug,
		Marketplace:     testMarketplace,
		Permissions:     models.PermissionRead | models.PermissionWrite,
	}
	testutil.Insert(t, db, testKey)

	gotKey, err := keyRepo.Read(testKey.ID)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if diff := cmp.Diff(*testKey, *gotKey, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Errorf("Diff: %s", diff)
	}
}

func TestKeyRepository_Read_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	_, err := keyRepo.Read("non-existent-key-id")
	if err == nil {
		t.Fatal("Read(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Read(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestKeyRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testutil.Insert(t, db, testMarketplace)

	testKeys := []*models.Key{
		{
			ID:              "test-key-id",
			Environment:     constants.EnvironmentProduction,
			MarketplaceSlug: testMarketplace.Slug,
			Permissions:     models.PermissionRead,
		},
		{
			ID:              "another-test-key-id",
			Environment:     constants.EnvironmentDevelopment,
			MarketplaceSlug: testMarketplace.Slug,
			Permissions:     models.PermissionWrite,
		},
	}
	testutil.Insert(t, db, testKeys...)

	if err := keyRepo.Delete(testKeys[0].ID); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var deletedKey models.Key
	err := db.Where("id = ?", testKeys[0].ID).First(&deletedKey).Error
	if err == nil {
		t.Fatal("expected record to be deleted, but it still exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}

	// Ensure the other key still exists
	var otherKey models.Key
	if err := db.Where("id = ?", testKeys[1].ID).First(&otherKey).Error; err != nil {
		t.Fatalf("expected other key to exist, but got error: %v", err)
	}

	if diff := cmp.Diff(*testKeys[1], otherKey, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Errorf("Diff: %s", diff)
	}
}

func TestKeyRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	err := keyRepo.Delete("non-existent-key-id")
	if err == nil {
		t.Fatal("Delete(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestKeyRepository_List(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplaces := []*models.Marketplace{
		{
			Slug: "marketplace-one",
			Name: "Marketplace One",
		},
		{
			Slug: "marketplace-two",
			Name: "Marketplace Two",
		},
	}
	testutil.Insert(t, db, testMarketplaces...)

	testKeys := []*models.Key{
		{
			ID:              "key-1",
			Environment:     constants.EnvironmentProduction,
			MarketplaceSlug: testMarketplaces[0].Slug,
			Permissions:     models.PermissionRead,
		},
		{
			ID:              "key-2",
			Environment:     constants.EnvironmentDevelopment,
			MarketplaceSlug: testMarketplaces[0].Slug,
			Permissions:     models.PermissionWrite,
		},
		{
			ID:              "key-3",
			Environment:     constants.EnvironmentProduction,
			MarketplaceSlug: testMarketplaces[1].Slug,
			Permissions:     models.PermissionManage,
		},
	}
	testutil.Insert(t, db, testKeys...)

	testCases := []struct {
		name string
		opts *dto.KeyListOptions
		want []*models.Key
	}{
		{
			name: "noFilters",
			opts: nil,
			want: testKeys,
		},
		{
			name: "filterByMarketplaceSlug",
			opts: &dto.KeyListOptions{
				MarketplaceSlug: &testMarketplaces[0].Slug,
			},
			want: []*models.Key{testKeys[0], testKeys[1]},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKeys, err := keyRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}

			sort.Slice(gotKeys, func(i, j int) bool {
				return gotKeys[i].ID < gotKeys[j].ID
			})

			if diff := cmp.Diff(tc.want, gotKeys, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt", "Marketplace")); diff != "" {
				t.Errorf("Diff: %s", diff)
			}
		})
	}
}
