package private

import (
	"errors"
	"sort"
	"testing"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/testutil"
	"reverse-watch/secret"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestKeyRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
		name        string
		slug        string
		permissions models.Permissions
	}{
		{
			name:        "validKey",
			slug:        "test-marketplace",
			permissions: models.PermissionWrite,
		},
		{
			name:        "adminKeyForCSFloat",
			slug:        "csfloat",
			permissions: models.PermissionAdmin,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rawKey, err := keyRepo.Create(tc.slug, tc.permissions)
			if err != nil {
				t.Fatalf("Create(): %v", err)
			}

			var storedKey models.Key
			if err := db.Where("id = ?", rawKey.ID).First(&storedKey).Error; err != nil {
				t.Fatalf("First(): %v", err)
			}

			if storedKey.CreatedAt == 0 {
				t.Errorf("got CreatedAt %d, wanted non-zero value", storedKey.CreatedAt)
			}
			if storedKey.UpdatedAt == 0 {
				t.Errorf("got UpdatedAt %d, wanted non-zero value", storedKey.UpdatedAt)
			}
		})
	}
}

func TestKeyRepository_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
		name        string
		slug        string
		permissions models.Permissions
		wantErr     string
	}{
		{
			name:        "emptyMarketplaceSlug",
			slug:        "",
			permissions: models.PermissionExport,
			wantErr:     "marketplace_slug is required",
		},
		{
			name:        "noPermissions",
			slug:        testMarketplace.Slug,
			permissions: models.PermissionNone,
			wantErr:     "at least one permission is required",
		},
		{
			name:        "adminPermissionNonCSFloat",
			slug:        "test-marketplace",
			permissions: models.PermissionAdmin,
			wantErr:     "admin scoped keys can only be created for CSFloat",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := keyRepo.Create(tc.slug, tc.permissions)
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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
		name        string
		slug        string
		permissions models.Permissions
	}{
		{
			name:        "singlePermission",
			slug:        testMarketplace.Slug,
			permissions: models.PermissionRead,
		},
		{
			name:        "multiplePermissions",
			slug:        testMarketplace.Slug,
			permissions: models.PermissionWrite | models.PermissionManage | models.PermissionExport,
		},
		{
			name:        "adminPermissionCSFloat",
			slug:        testCSFloatMarketplace.Slug,
			permissions: models.PermissionAdmin,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rawKey, err := keyRepo.Create(tc.slug, tc.permissions)
			if err != nil {
				t.Fatalf("Create(): %v", err)
			}

			if rawKey.ID == "" {
				t.Error("got empty id")
			}
			if rawKey.SecretKey == "" {
				t.Error("got empty secret key")
			}
			if rawKey.MarketplaceSlug != tc.slug {
				t.Errorf("got slug %q, wanted %q", rawKey.MarketplaceSlug, tc.slug)
			}
			if rawKey.Permissions != tc.permissions {
				t.Errorf("got permissions %d, wanted %d", rawKey.Permissions, tc.permissions)
			}
			if rawKey.Environment != constants.EnvironmentDevelopment {
				t.Errorf("got environment %q, wanted %q", rawKey.Environment, constants.EnvironmentDevelopment)
			}

			var storedKey models.Key
			if err := db.Where("id = ?", rawKey.ID).First(&storedKey).Error; err != nil {
				t.Fatalf("failed to retrieve stored key: %v", err)
			}

			if storedKey.CreatedAt == 0 {
				t.Errorf("got CreatedAt %d, wanted non-zero value", storedKey.CreatedAt)
			}
			if storedKey.UpdatedAt == 0 {
				t.Errorf("got UpdatedAt %d, wanted non-zero value", storedKey.UpdatedAt)
			}
			if storedKey.ID != rawKey.ID {
				t.Errorf("ID mismatch: got storedKey ID %q, rawKey ID %q", storedKey.ID, rawKey.ID)
			}
			if storedKey.Environment != rawKey.Environment {
				t.Errorf("Environment mismatch: got storedKey Environment %q, rawKey Environment %q", storedKey.Environment, rawKey.Environment)
			}
			if storedKey.MarketplaceSlug != rawKey.MarketplaceSlug {
				t.Errorf("MarketplaceSlug mismatch: got storedKey MarketplaceSlug %q, rawKey MarketplaceSlug %q", storedKey.MarketplaceSlug, rawKey.MarketplaceSlug)
			}
			if storedKey.Permissions != rawKey.Permissions {
				t.Errorf("Permissions mismatch: got storedKey Permissions %q, rawKey Permissions %q", storedKey.Permissions, rawKey.Permissions)
			}
		})
	}
}

func TestKeyRepository_Create_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testutil.Insert(t, db, testMarketplace)

	testCases := []struct {
		name        string
		slug        string
		permissions models.Permissions
		wantErr     string
	}{
		{
			name:        "nonExistentMarketplace",
			slug:        "non-existent-marketplace",
			permissions: models.PermissionRead,
			wantErr:     "FOREIGN KEY constraint failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := keyRepo.Create(tc.slug, tc.permissions)
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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

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

func TestKeyRepository_ValidateKey(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testutil.Insert(t, db, testMarketplace)

	secretKey, err := keygen.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey(): %v", err)
	}

	id, err := secretKey.ID()
	if err != nil {
		t.Fatalf("ID(): %v", err)
	}

	testKey := &models.Key{
		ID:              id,
		Environment:     keygen.Environment(),
		MarketplaceSlug: testMarketplace.Slug,
		Marketplace:     testMarketplace,
		Permissions:     models.PermissionRead,
	}
	testutil.Insert(t, db, testKey)

	formattedKey, err := secretKey.Format()
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}

	gotKey, err := keyRepo.ValidateKey(formattedKey)
	if err != nil {
		t.Fatalf("ValidateKey(): %v", err)
	}
	if diff := cmp.Diff(*testKey, *gotKey, cmpopts.IgnoreFields(models.Key{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Errorf("Diff: %s", diff)
	}
}

func TestKeyRepository_ValidateKey_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	keygen := secret.NewKeyGenerator(constants.EnvironmentDevelopment)
	keyRepo := NewKeyRepository(db, keygen)

	testMarketplace := &models.Marketplace{
		Slug: "test-marketplace",
		Name: "Test Marketplace",
	}
	testutil.Insert(t, db, testMarketplace)

	secretKey := "non-existent-key"
	_, err := keyRepo.ValidateKey(secretKey)
	if err == nil {
		t.Fatal("ValidateKey(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("ValidateKey(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}
