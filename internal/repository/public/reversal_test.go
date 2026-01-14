package public

import (
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestReversalRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testCases := []struct {
		name     string
		reversal *models.Reversal
	}{
		{
			name: "validReversal",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := reversalRepo.Create(tc.reversal); err != nil {
				t.Fatalf("Create(): %v", err)
			}
			if tc.reversal.CreatedAt == 0 {
				t.Errorf("got CreatedAt %d, wanted non-zero value", tc.reversal.CreatedAt)
			}
			if tc.reversal.UpdatedAt == 0 {
				t.Errorf("got UpdatedAt %d, wanted non-zero value", tc.reversal.UpdatedAt)
			}
			if tc.reversal.ReversedAt == 0 {
				t.Errorf("got ReversedAt %d, wanted non-zero value", tc.reversal.ReversedAt)
			}
		})
	}
}

func TestReversalRepository_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testCases := []struct {
		name     string
		reversal *models.Reversal
		wantErr  string
	}{
		{
			name: "invalidSteamID",
			reversal: &models.Reversal{
				SteamID: 0,
			},
			wantErr: "steam_id is invalid",
		},
		{
			name: "emptyMarketplaceSlug",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "",
			},

			wantErr: "marketplace_slug is required",
		},
		{
			name: "invalidRelatedSteamID",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				RelatedSteamID:  testutil.Ptr(models.SteamID(0)),
			},
			wantErr: "related_steam_id is invalid",
		},
		{
			name: "validRelatedSteamIDWithInvalidSource",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          testutil.Ptr(models.SourceDirect),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "validSourceWithNilRelatedSteamID",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  nil,
			},
			wantErr: "related_steam_id is required when source is \"related_user\"",
		},
		{
			name: "invalidReversedAt",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ReversedAt:      uint64(99999999999999999),
			},
			wantErr: "reversed_at cannot be in the future",
		},
		{
			name: "invalidExpungedAt",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ExpungedAt:      testutil.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at cannot be in the future",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reversalRepo.Create(tc.reversal)
			if err == nil {
				t.Fatalf("Create(): got nil error, wanted %s", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("Create(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}

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
