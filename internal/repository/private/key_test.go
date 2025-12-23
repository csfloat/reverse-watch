package private

import (
	"reflect"
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/testutil"
)

func TestKeyRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	keyRepo := NewKeyRepository(db)

	id, err := models.GenSnowflake()
	if err != nil {
		t.Fatal(err)
	}

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	testKey := &models.Key{
		Model: models.Model{
			ID: id,
		},
		KeyHash:         "test-hash",
		Salt:            "test-salt",
		MarketplaceSlug: testMarketplace.Slug,
		Permissions:     models.PermissionNone,
	}

	if err := keyRepo.Create(testKey); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	var gotKey models.Key
	if err := db.First(&gotKey, "id = ?", id).Error; err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if gotKey.KeyHash != testKey.KeyHash {
		t.Errorf("got KeyHash %s, wanted %s", gotKey.KeyHash, testKey.KeyHash)
	}
	if gotKey.Salt != testKey.Salt {
		t.Errorf("got Salt %s, wanted %s", gotKey.Salt, testKey.Salt)
	}
	if gotKey.MarketplaceSlug != testKey.MarketplaceSlug {
		t.Errorf("got MarketplaceSlug %s, wanted %s", gotKey.MarketplaceSlug, testKey.MarketplaceSlug)
	}
	if gotKey.Permissions != testKey.Permissions {
		t.Errorf("got Permissions %v, wanted %v", gotKey.Permissions, testKey.Permissions)
	}
}

func TestKeyRepository_Create_Error(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	testCases := []struct {
		name    string
		key     *models.Key
		wantErr string
	}{
		{
			name: "nonExistentMarketplace",
			key: &models.Key{
				Model: models.Model{
					ID: 5,
				},
				KeyHash:         "test-hash",
				Salt:            "test-salt",
				MarketplaceSlug: "non-existent-marketplace",
			},
			wantErr: "FOREIGN KEY constraint failed",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := keyRepo.Create(tc.key)
			if err == nil {
				t.Fatalf("Create(): got nil error, wanted %s", tc.wantErr)
			}

			if err.Error() != tc.wantErr {
				t.Errorf("Create(): got error %v, wanted %s", err, tc.wantErr)
			}
		})
	}
}

func TestKeyRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	testKey := &models.Key{
		Model: models.Model{
			ID: 1,
		},
		KeyHash:         "test-hash",
		Salt:            "test-salt",
		MarketplaceSlug: testMarketplace.Slug,
	}
	testutil.Insert(t, db, testKey)

	gotKey, err := keyRepo.Read(testKey.ID)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if gotKey.KeyHash != testKey.KeyHash {
		t.Errorf("got KeyHash %s, wanted %s", gotKey.KeyHash, testKey.KeyHash)
	}
	if gotKey.Salt != testKey.Salt {
		t.Errorf("got Salt %s, wanted %s", gotKey.Salt, testKey.Salt)
	}
	if gotKey.MarketplaceSlug != testKey.MarketplaceSlug {
		t.Errorf("got MarketplaceSlug %s, wanted %s", gotKey.MarketplaceSlug, testKey.MarketplaceSlug)
	}
	if gotKey.Permissions != testKey.Permissions {
		t.Errorf("got Permissions %v, wanted %v", gotKey.Permissions, testKey.Permissions)
	}

	// Ensure marketplace was preloaded
	if gotKey.Marketplace == nil {
		t.Fatalf("got nil marketplace, wanted %v", testMarketplace)
	}
	if gotKey.Marketplace.Slug != testMarketplace.Slug {
		t.Errorf("got Marketplace.Slug %s, wanted %s", gotKey.Marketplace.Slug, testMarketplace.Slug)
	}
	if gotKey.Marketplace.Name != testMarketplace.Name {
		t.Errorf("got Marketplace.Name %s, wanted %s", gotKey.Marketplace.Name, testMarketplace.Name)
	}
	if gotKey.Marketplace.IsActive != testMarketplace.IsActive {
		t.Errorf("got Marketplace.IsActive %v, wanted %v", gotKey.Marketplace.IsActive, testMarketplace.IsActive)
	}
}

func TestKeyRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	testKey := &models.Key{
		Model: models.Model{
			ID: 1,
		},
		KeyHash:         "test-hash",
		Salt:            "test-salt",
		MarketplaceSlug: testMarketplace.Slug,
	}
	testutil.Insert(t, db, testKey)

	err := keyRepo.Delete(testKey.ID)
	if err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var deletedKey models.Key
	if err := db.Where("id = ?", 2).First(&deletedKey).Error; err == nil {
		t.Fatalf("Delete(): got nil error, wanted not found error")
	}
}

func TestKeyRepository_List(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	keyRepo := NewKeyRepository(db)

	testMarketplace1 := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testMarketplace2 := &models.Marketplace{
		Slug:     "test-marketplace-2",
		Name:     "Test Marketplace 2",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace1, testMarketplace2)

	testKeys := []*models.Key{
		{
			Model: models.Model{
				ID: 1,
			},
			KeyHash:         "test-hash-1",
			Salt:            "test-salt-1",
			MarketplaceSlug: testMarketplace1.Slug,
		},
		{
			Model: models.Model{
				ID: 2,
			},
			KeyHash:         "test-hash-2",
			Salt:            "test-salt-2",
			MarketplaceSlug: testMarketplace1.Slug,
		},
		{
			Model: models.Model{
				ID: 3,
			},
			KeyHash:         "test-hash-3",
			Salt:            "test-salt-3",
			MarketplaceSlug: testMarketplace2.Slug,
		},
		{
			Model: models.Model{
				ID: 4,
			},
			KeyHash:         "test-hash-4",
			Salt:            "test-salt-4",
			MarketplaceSlug: testMarketplace2.Slug,
		},
	}
	testutil.Insert(t, db, testKeys...)

	testCases := []struct {
		name     string
		opts     *repository.KeyListOptions
		wantKeys []*models.Key
	}{
		{
			name:     "noOptions",
			opts:     nil,
			wantKeys: testKeys,
		},
		{
			name: "marketplaceFilter",
			opts: &repository.KeyListOptions{
				MarketplaceSlug: &testMarketplace1.Slug,
			},
			wantKeys: testKeys[:2],
		},
		{
			name: "marketplaceFilterNotFound",
			opts: &repository.KeyListOptions{
				MarketplaceSlug: testutil.Ptr("non-existent-marketplace"),
			},
			wantKeys: []*models.Key{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotKeys, err := keyRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}

			if !reflect.DeepEqual(gotKeys, tc.wantKeys) {
				t.Errorf("got keys %v, wanted %v", gotKeys, tc.wantKeys)
			}
		})
	}
}
