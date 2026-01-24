package marketplace

import (
	"errors"
	"testing"
	"time"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/mocks"
	"reverse-watch/internal/util"

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
	if testMarketplace.CreatedAt == 0 {
		t.Errorf("got CreatedAt %d, wanted non-zero value", testMarketplace.CreatedAt)
	}
	if testMarketplace.UpdatedAt == 0 {
		t.Errorf("got UpdatedAt %d, wanted non-zero value", testMarketplace.UpdatedAt)
	}
	if testMarketplace.Slug != "test-marketplace" {
		t.Errorf("got Slug %q, wanted %q", testMarketplace.Slug, "test-marketplace")
	}
	if testMarketplace.Name != "Test Marketplace" {
		t.Errorf("got Name %q, wanted %q", testMarketplace.Name, "Test Marketplace")
	}
	if testMarketplace.IsActive != true {
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
