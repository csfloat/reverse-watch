package public

import (
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestReversalRepository_Create(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testCases := []struct {
		name      string
		reversals []*models.Reversal
	}{
		{
			name: "singleReversal",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
				},
			},
		},
		{
			name: "multipleReversals",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
				},
				{
					SteamID:         models.SteamID(76561197960287931),
					MarketplaceSlug: "test-slug-2",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := reversalRepo.Create(tc.reversals...); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			var ids []models.Snowflake
			for _, reversal := range tc.reversals {
				ids = append(ids, reversal.ID)
			}

			var storedReversals []*models.Reversal
			if err := db.Where("id IN (?)", ids).Find(&storedReversals).Error; err != nil {
				t.Fatalf("failed to retrieve stored reversals: %v", err)
			}

			if diff := cmp.Diff(storedReversals, tc.reversals, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}
