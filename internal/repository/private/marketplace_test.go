package private

import (
	"errors"
	"testing"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestMarketplaceRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testCases := []struct {
		name        string
		marketplace *models.Marketplace
	}{
		{
			name: "validMarketplace",
			marketplace: &models.Marketplace{
				Slug:     "test-marketplace",
				Name:     "Test Marketplace",
				IsActive: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := marketplaceRepo.Create(tc.marketplace); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			if tc.marketplace.CreatedAt == 0 {
				t.Errorf("got CreatedAt %d, wanted non-zero value", tc.marketplace.CreatedAt)
			}
			if tc.marketplace.UpdatedAt == 0 {
				t.Errorf("got UpdatedAt %d, wanted non-zero value", tc.marketplace.UpdatedAt)
			}
		})
	}
}

func TestMarketplaceRepository_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testCases := []struct {
		name        string
		marketplace *models.Marketplace
		wantErr     string
	}{
		{
			name: "emptySlug",
			marketplace: &models.Marketplace{
				Slug: "",
				Name: "Test Marketplace",
			},
			wantErr: "slug must be between 1 and 25 characters long",
		},
		{
			name: "invalidSlug",
			marketplace: &models.Marketplace{
				Slug: "slug with spaces",
				Name: "Test Marketplace",
			},
			wantErr: "slug must contain only letters, numbers, and hyphens",
		},
		{
			name: "emptyName",
			marketplace: &models.Marketplace{
				Slug: "test-marketplace",
				Name: "",
			},
			wantErr: "name must be between 1 and 50 characters long",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := marketplaceRepo.Create(tc.marketplace)
			if err == nil {
				t.Fatalf("Create(): expected error")
			}

			if err.Error() != tc.wantErr {
				t.Errorf("got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}

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

func TestMarketplaceRepository_Create_DuplicatePrimaryKey(t *testing.T) {
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
	err := marketplaceRepo.Create(testMarketplace)
	if err == nil {
		t.Fatalf("Create(): got nil error, wanted error")
	}
	if err.Error() != "UNIQUE constraint failed: marketplaces.slug" {
		t.Fatalf("Create(): got error %v, wanted %v", err, "UNIQUE constraint failed: marketplaces.slug")
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

	_, err := marketplaceRepo.Read("non-existent-slug")
	if err == nil {
		t.Fatalf("Read(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Read(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
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
	testutil.Insert(t, db, testMarketplace1, testMarketplace2)

	testCases := []struct {
		name     string
		original *models.Marketplace
		opts     *dto.MarketplaceUpdateOptions
		want     *models.Marketplace
	}{
		{
			name:     "allFields",
			original: testMarketplace1,
			opts: &dto.MarketplaceUpdateOptions{
				Name:     testutil.Ptr("Updated Test Marketplace 1"),
				IsActive: testutil.Ptr(false),
			},
			want: &models.Marketplace{
				Slug:     "test-marketplace1",
				Name:     "Updated Test Marketplace 1",
				IsActive: false,
			},
		},
		{
			name:     "partialFields",
			original: testMarketplace2,
			opts: &dto.MarketplaceUpdateOptions{
				Name: testutil.Ptr("Updated Test Marketplace 2"),
			},
			want: &models.Marketplace{
				Slug:     "test-marketplace2",
				Name:     "Updated Test Marketplace 2",
				IsActive: true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := marketplaceRepo.Update(tc.original.Slug, tc.opts); err != nil {
				t.Fatalf("Update(): %v", err)
			}

			var gotMarketplace *models.Marketplace
			if err := db.Where("slug = ?", tc.original.Slug).First(&gotMarketplace).Error; err != nil {
				t.Fatalf("First(): %v", err)
			}

			if diff := cmp.Diff(gotMarketplace, tc.want, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestMarketplaceRepository_Update_Errors(t *testing.T) {
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
	testutil.Insert(t, db, testMarketplace1, testMarketplace2)

	testCases := []struct {
		name    string
		slug    string
		opts    *dto.MarketplaceUpdateOptions
		wantErr string
	}{
		{
			name:    "noOptions",
			slug:    testMarketplace1.Slug,
			opts:    &dto.MarketplaceUpdateOptions{},
			wantErr: "marketplace update options is empty",
		},
		{
			name:    "nilOptions",
			slug:    testMarketplace2.Slug,
			opts:    nil,
			wantErr: "marketplace update options cannot be nil",
		},
		{
			name: "recordNotFound",
			slug: "test-marketplace3",
			opts: &dto.MarketplaceUpdateOptions{
				Name:     testutil.Ptr("Updated Test Marketplace 3"),
				IsActive: testutil.Ptr(false),
			},
			wantErr: "record not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := marketplaceRepo.Update(tc.slug, tc.opts)
			if err == nil {
				t.Fatalf("Update(): got nil error, wanted error")
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("got error: %v, wanted error: %v", err, tc.wantErr)
			}
		})
	}
}

func TestMarketplaceRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testMarketplaces := []*models.Marketplace{
		{
			Slug:     "test-marketplace-1",
			Name:     "Test Marketplace 1",
			IsActive: true,
		},
		{
			Slug:     "test-marketplace-2",
			Name:     "Test Marketplace 2",
			IsActive: true,
		},
	}
	testutil.Insert(t, db, testMarketplaces...)

	testKeys := []*models.Key{
		{
			ID:              "test-key-hash-1",
			Environment:     models.EnvironmentProduction,
			MarketplaceSlug: testMarketplaces[0].Slug,
		},
		{
			ID:              "test-key-hash-2",
			Environment:     models.EnvironmentDevelopment,
			MarketplaceSlug: testMarketplaces[1].Slug,
		},
	}
	testutil.Insert(t, db, testKeys...)

	if err := marketplaceRepo.Delete(testMarketplaces[0].Slug); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var deletedMarketplace models.Marketplace
	err := db.Where("slug = ?", testMarketplaces[0].Slug).First(&deletedMarketplace).Error
	if err == nil {
		t.Fatalf("got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("got %v, wanted %v", err, gorm.ErrRecordNotFound)
	}

	// Ensure corresponding keys were deleted
	var deletedKey models.Key
	err = db.Where("marketplace_slug = ?", testMarketplaces[0].Slug).First(&deletedKey).Error
	if err == nil {
		t.Fatalf("got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("got %v, wanted %v", err, gorm.ErrRecordNotFound)
	}

	// Ensure other marketplace remains
	var otherMarketplace models.Marketplace
	if err := db.Where("slug = ?", testMarketplaces[1].Slug).First(&otherMarketplace).Error; err != nil {
		t.Fatalf("First(): failed to fetch remaining marketplace %v", err)
	}

	if diff := cmp.Diff(&otherMarketplace, testMarketplaces[1]); diff != "" {
		t.Error(diff)
	}

	// Ensure other marketplace's keys remain
	var otherKey models.Key
	if err := db.Where("marketplace_slug = ?", testMarketplaces[1].Slug).First(&otherKey).Error; err != nil {
		t.Fatalf("First(): failed to fetch remaining marketplace's keys %v", err)
	}

	if diff := cmp.Diff(&otherKey, testKeys[1]); diff != "" {
		t.Error(diff)
	}
}

func TestMarketplaceRepository_Delete_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewPrivateTestDB(t)
	marketplaceRepo := NewMarketplaceRepository(db)

	testCases := []struct {
		name    string
		slug    string
		wantErr string
	}{
		{
			name:    "recordNotFound",
			slug:    "non-existent-marketplace",
			wantErr: "record not found",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := marketplaceRepo.Delete(tc.slug)
			if err == nil {
				t.Fatalf("Delete(): got nil error, wanted error")
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("got error: %v, wanted error: %v", err, tc.wantErr)
			}
		})
	}
}
