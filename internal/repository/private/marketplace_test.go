package private

import (
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"
)

func TestMarketplaceRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	if err := marketplaceRepo.Create(testMarketplace); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	var createdMarketplace models.Marketplace
	if err := db.Where("slug = ?", testMarketplace.Slug).First(&createdMarketplace).Error; err != nil {
		t.Errorf("First(): %v", err)
	}
}

func TestMarketplaceRepository_Create_Error(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	// Attempt to create the same marketplace again
	if err := marketplaceRepo.Create(testMarketplace); err == nil {
		t.Fatalf("Create(): expected error")
	}
}

func TestMarketplaceRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	gotMarketplace, err := marketplaceRepo.Read(testMarketplace.Slug)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if gotMarketplace.Slug != testMarketplace.Slug {
		t.Errorf("got slug %q, wanted %q", gotMarketplace.Slug, testMarketplace.Slug)
	}
	if gotMarketplace.Name != testMarketplace.Name {
		t.Errorf("got name %q, wanted %q", gotMarketplace.Name, testMarketplace.Name)
	}
	if gotMarketplace.IsActive != testMarketplace.IsActive {
		t.Errorf("got isActive %v, wanted %v", gotMarketplace.IsActive, testMarketplace.IsActive)
	}
}

func TestMarketplaceRepository_Update(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	fieldsToUpdate := map[string]interface{}{
		"slug":      "updated-test-marketplace",
		"name":      "Updated Test Marketplace",
		"is_active": false,
	}
	if err := marketplaceRepo.Update(testMarketplace.Slug, fieldsToUpdate); err != nil {
		t.Fatalf("Update(): %v", err)
	}

	var updatedMarketplace models.Marketplace
	if err := db.Where("slug = ?", fieldsToUpdate["slug"]).First(&updatedMarketplace).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}

	if updatedMarketplace.Slug != fieldsToUpdate["slug"] {
		t.Errorf("got slug %q, wanted %q", updatedMarketplace.Slug, fieldsToUpdate["slug"])
	}
	if updatedMarketplace.Name != fieldsToUpdate["name"] {
		t.Errorf("got name %q, wanted %q", updatedMarketplace.Name, fieldsToUpdate["name"])
	}
	if updatedMarketplace.IsActive != fieldsToUpdate["is_active"] {
		t.Errorf("got isActive %v, wanted %v", updatedMarketplace.IsActive, fieldsToUpdate["is_active"])
	}
}

func TestMarketplaceRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}
	testutil.Insert(t, db, testMarketplace)

	if err := marketplaceRepo.Delete(testMarketplace.Slug); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var deletedMarketplace models.Marketplace
	if err := db.Where("slug = ?", testMarketplace.Slug).First(&deletedMarketplace).Error; err == nil {
		t.Fatalf("got nil error, wanted error")
	}
}
