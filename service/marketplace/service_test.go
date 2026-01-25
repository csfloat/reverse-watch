package marketplace

import (
	"errors"
	"testing"
	"time"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/mocks"
	"reverse-watch/util"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestMarketplaceService_CreateMarketplace(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	err := marketplaceSvc.CreateMarketplace(testMarketplace)
	if err != nil {
		t.Fatalf("CreateMarketplace(): %v", err)
	}

	gotMarketplace, err := mockPrivateRepo.Marketplace().Read(testMarketplace.Slug)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if gotMarketplace.CreatedAt == 0 {
		t.Errorf("got CreatedAt %d, wanted non-zero value", gotMarketplace.CreatedAt)
	}
	if gotMarketplace.UpdatedAt == 0 {
		t.Errorf("got UpdatedAt %d, wanted non-zero value", gotMarketplace.UpdatedAt)
	}
	if gotMarketplace.Slug != "test-marketplace" {
		t.Errorf("got Slug %q, wanted %q", gotMarketplace.Slug, "test-marketplace")
	}
	if gotMarketplace.Name != "Test Marketplace" {
		t.Errorf("got Name %q, wanted %q", gotMarketplace.Name, "Test Marketplace")
	}
	if gotMarketplace.IsActive != true {
		t.Errorf("got IsActive %v, wanted %v", testMarketplace.IsActive, true)
	}
}

func TestMarketplaceService_CreateMarketplace_DuplicateSlug(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	err := marketplaceSvc.CreateMarketplace(testMarketplace)
	if err != nil {
		t.Fatalf("CreateMarketplace(): %v", err)
	}

	err = marketplaceSvc.CreateMarketplace(testMarketplace)
	if err == nil {
		t.Fatalf("CreateMarketplace(): got nil error, wanted error")
	}
	if err.Error() != "UNIQUE constraint failed: marketplaces.slug" {
		t.Fatalf("CreateMarketplace(): got error %v, wanted %v", err, "UNIQUE constraint failed: marketplaces.slug")
	}
}

func TestMarketplaceService_GetMarketplace(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	if err := mockPrivateRepo.Marketplace().Create(testMarketplace); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	gotMarketplace, err := marketplaceSvc.GetMarketplace(testMarketplace.Slug)
	if err != nil {
		t.Fatalf("GetMarketplace(): %v", err)
	}

	if diff := cmp.Diff(testMarketplace, gotMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Error(diff)
	}
}

func TestMarketplaceService_GetMarketplace_NotFound(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	_, err := marketplaceSvc.GetMarketplace("test-marketplace")
	if err == nil {
		t.Fatalf("GetMarketplace(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("GetMarketplace(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestMarketplaceService_UpdateMarketplace(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	now := uint64(time.Now().UnixMilli())
	testCases := []struct {
		name     string
		original *models.Marketplace
		updates  *dto.MarketplaceUpdates
		want     *models.Marketplace
	}{
		{
			name: "allFields",
			original: &models.Marketplace{
				Slug:      "test-marketplace-1",
				CreatedAt: now,
				UpdatedAt: now,
				Name:      "Test Marketplace 1",
				IsActive:  true,
			},
			updates: &dto.MarketplaceUpdates{
				Name:     util.Ptr("Updated Test Marketplace 1"),
				IsActive: util.Ptr(false),
			},
			want: &models.Marketplace{
				Slug:      "test-marketplace-1",
				CreatedAt: now,
				Name:      "Updated Test Marketplace 1",
				IsActive:  false,
			},
		},
		{
			name: "partialFields",
			original: &models.Marketplace{
				Slug:      "test-marketplace-2",
				CreatedAt: now,
				UpdatedAt: now,
				Name:      "Test Marketplace 2",
				IsActive:  true,
			},
			updates: &dto.MarketplaceUpdates{
				Name: util.Ptr("Updated Test Marketplace 2"),
			},
			want: &models.Marketplace{
				Slug:      "test-marketplace-2",
				CreatedAt: now,
				Name:      "Updated Test Marketplace 2",
				IsActive:  true,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := mockPrivateRepo.Marketplace().Create(tc.original); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			if err := marketplaceSvc.UpdateMarketplace(tc.original.Slug, tc.updates); err != nil {
				t.Fatalf("UpdateMarketplace(): %v", err)
			}

			gotMarketplace, err := mockPrivateRepo.Marketplace().Read(tc.original.Slug)
			if err != nil {
				t.Fatalf("Read(): %v", err)
			}

			if diff := cmp.Diff(tc.want, gotMarketplace, cmpopts.IgnoreFields(models.Marketplace{}, "UpdatedAt")); diff != "" {
				t.Error(diff)
			}

			// Ensure UpdatedAt was updated
			if gotMarketplace.UpdatedAt < tc.want.UpdatedAt {
				t.Errorf("got UpdatedAt %v, wanted time greater than %v", gotMarketplace.UpdatedAt, now)
			}
		})
	}
}

func TestMarketplaceService_UpdateMarketplace_Errors(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	if err := mockPrivateRepo.Marketplace().Create(testMarketplace); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	updates := &dto.MarketplaceUpdates{
		Name: util.Ptr(""),
	}
	wantErr := "invalid marketplace updates: name must be between 1 and 50 characters long"

	err := marketplaceSvc.UpdateMarketplace(testMarketplace.Slug, updates)
	if err == nil {
		t.Fatalf("UpdateMarketplace(): got nil error, wanted %q", wantErr)
	}
	if err.Error() != wantErr {
		t.Fatalf("UpdateMarketplace(): got error %q, wanted %q", err, wantErr)
	}
}

func TestMarketplaceService_DeleteMarketplace(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	testMarketplace := &models.Marketplace{
		Slug:     "test-marketplace",
		Name:     "Test Marketplace",
		IsActive: true,
	}

	if err := mockPrivateRepo.Marketplace().Create(testMarketplace); err != nil {
		t.Fatalf("Create(): %v", err)
	}

	if err := marketplaceSvc.DeleteMarketplace(testMarketplace.Slug); err != nil {
		t.Fatalf("DeleteMarketplace(): %v", err)
	}

	_, err := mockPrivateRepo.Marketplace().Read(testMarketplace.Slug)
	if err == nil {
		t.Fatalf("DeleteMarketplace(): got nil error, wanted %v", gorm.ErrRecordNotFound)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("DeleteMarketplace(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestMarketplaceService_DeleteMarketplace_NotFound(t *testing.T) {
	t.Parallel()

	mockPrivateRepo := mocks.NewMockPrivateRepository()
	marketplaceSvc := NewMarketplaceService(mockPrivateRepo)

	err := marketplaceSvc.DeleteMarketplace("test-marketplace")
	if err == nil {
		t.Fatalf("DeleteMarketplace(): got nil error, wanted %v", gorm.ErrRecordNotFound)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("DeleteMarketplace(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}
