package private

import (
	"errors"
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
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

func TestMarketplaceRepository_Read_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	_, err := marketplaceRepo.Read("not-existent-slug")
	if err == nil || !errors.Is(gorm.ErrRecordNotFound, err) {
		t.Fatalf("got nil error, wanted %v error", gorm.ErrRecordNotFound)
	}
}

func TestMarketplaceRepository_Update(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplace1 := &models.Marketplace{
		Slug:     "test-marketplace1",
		Name:     "Test Marketplace 1",
		IsActive: true,
	}
	testMarketplace2 := &models.Marketplace{
		Slug:     "test-marketplace2",
		Name:     "Test Marketplace 2",
		IsActive: true,
	}
	testMarketplace3 := &models.Marketplace{
		Slug:     "test-marketplace3",
		Name:     "Test Marketplace 3",
		IsActive: false,
	}

	testCases := []struct {
		name     string
		original *models.Marketplace
		fields   map[string]interface{}
		want     *models.Marketplace
	}{
		{
			name:     "allFields",
			original: testMarketplace1,
			fields: map[string]interface{}{
				"slug":      "updated-test-marketplace1",
				"name":      "Updated Test Marketplace 1",
				"is_active": false,
			},
			want: &models.Marketplace{
				Slug:     "updated-test-marketplace1",
				Name:     "Updated Test Marketplace 1",
				IsActive: false,
			},
		},
		{
			name:     "partialFields",
			original: testMarketplace2,
			fields: map[string]interface{}{
				"name": "Updated Test Marketplace 2",
			},
			want: &models.Marketplace{
				Slug:     "test-marketplace2",
				Name:     "Updated Test Marketplace 2",
				IsActive: true,
			},
		},
		{
			name:     "noFields",
			original: testMarketplace3,
			fields:   map[string]interface{}{},
			want:     testMarketplace3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.Insert(t, db, tc.original)

			if err := marketplaceRepo.Update(tc.original.Slug, tc.fields); err != nil {
				t.Fatalf("Update(): %v", err)
			}

			slug := tc.original.Slug
			if updatedSlug, ok := tc.fields["slug"].(string); ok {
				slug = updatedSlug
			}

			var gotMarketplace models.Marketplace
			if err := db.Where("slug = ?", slug).First(&gotMarketplace).Error; err != nil {
				t.Fatalf("First(): %v", err)
			}

			if diff := cmp.Diff(gotMarketplace, *tc.want, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
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
