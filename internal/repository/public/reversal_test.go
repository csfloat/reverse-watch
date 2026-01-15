package public

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
		{
			name: "validSourceAndRelatedSteamID",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
					Source:          testutil.Ptr(models.SourceRelatedUser),
					RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
				},
			},
		},
		{
			name: "sourceDirect",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
					Source:          testutil.Ptr(models.SourceDirect),
				},
			},
		},
		{
			name: "sourceUserReport",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
					Source:          testutil.Ptr(models.SourceUserReport),
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

func TestReversalRepository_Create_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testCases := []struct {
		name     string
		reversal *models.Reversal
		wantErr  error
	}{
		{
			name:     "nilReversal",
			reversal: nil,
			wantErr:  gorm.ErrInvalidValue,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reversalRepo.Create(tc.reversal)
			if err == nil {
				t.Fatalf("Create(): got nil error, wanted error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("Create(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}

func TestReversalRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
		Source:          testutil.Ptr(models.SourceRelatedUser),
		RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
		ExpungedAt:      testutil.Ptr(uint64(1)),
	}
	testutil.Insert(t, db, testReversal)

	gotReversal, err := reversalRepo.Read(testReversal.ID)
	if err != nil {
		t.Fatalf("Read(): %v", err)
	}

	if diff := cmp.Diff(testReversal, gotReversal, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt")); diff != "" {
		t.Error(diff)
	}
}

func TestReversalRepository_Update(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
	}
	testutil.Insert(t, db, testReversal)

	testCases := []struct {
		name string
		opts *dto.ReversalUpdateOptions
		want *models.Reversal
	}{
		{
			name: "steamId",
			opts: &dto.ReversalUpdateOptions{
				SteamID: testutil.Ptr(models.SteamID(76561197960287932)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: testReversal.MarketplaceSlug,
				ReversedAt:      testReversal.ReversedAt,
			},
		},
		{
			name: "marketplaceSlug",
			opts: &dto.ReversalUpdateOptions{
				MarketplaceSlug: testutil.Ptr("updated-slug"),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				ReversedAt:      testReversal.ReversedAt,
			},
		},
		{
			name: "source",
			opts: &dto.ReversalUpdateOptions{
				Source: testutil.Ptr(models.SourceDirect),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				Source:          testutil.Ptr(models.SourceDirect),
				ReversedAt:      testReversal.ReversedAt,
			},
		},
		{
			name: "relatedUser",
			opts: &dto.ReversalUpdateOptions{
				Source:         testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID: testutil.Ptr(models.SteamID(76561197960287931)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
				ReversedAt:      testReversal.ReversedAt,
			},
		},
		{
			name: "relatedSteamId",
			opts: &dto.ReversalUpdateOptions{
				RelatedSteamID: testutil.Ptr(models.SteamID(76561197960287933)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287933)),
				ReversedAt:      testReversal.ReversedAt,
			},
		},
		{
			name: "reversedAt",
			opts: &dto.ReversalUpdateOptions{
				ReversedAt: testutil.Ptr(uint64(1)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287933)),
				ReversedAt:      uint64(1),
			},
		},
		{
			name: "expungedAt",
			opts: &dto.ReversalUpdateOptions{
				ExpungedAt: testutil.Ptr(uint64(1)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: testReversal.ID,
				},
				SteamID:         models.SteamID(76561197960287932),
				MarketplaceSlug: "updated-slug",
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287933)),
				ReversedAt:      uint64(1),
				ExpungedAt:      testutil.Ptr(uint64(1)),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := reversalRepo.Update(testReversal.ID, tc.opts); err != nil {
				t.Fatalf("Update(): %v", err)
			}

			var got models.Reversal
			if err := db.Where("id = ?", testReversal.ID).First(&got).Error; err != nil {
				t.Fatalf("First(): %v", err)
			}

			if diff := cmp.Diff(tc.want, &got, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalRepository_Update_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewPublicTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
	}
	testutil.Insert(t, db, testReversal)

	testCases := []struct {
		name    string
		opts    *dto.ReversalUpdateOptions
		wantErr string
	}{
		{
			name:    "nilOptions",
			opts:    nil,
			wantErr: "opts cannot be nil",
		},
		{
			name:    "emptyOptions",
			opts:    &dto.ReversalUpdateOptions{},
			wantErr: "no fields to update",
		},
		{
			name: "emptyMarketplaceSlug",
			opts: &dto.ReversalUpdateOptions{
				MarketplaceSlug: testutil.Ptr(""),
			},
			wantErr: "cannot set an empty marketplace_slug",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reversalRepo.Update(testReversal.ID, tc.opts)
			if err == nil {
				t.Fatalf("Update(): got nil error, wanted error")
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Update(): got error %s, wanted %s", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestReversalRepository_Update_Error_InvalidSourceAndRelatedSteamId(t *testing.T) {
	t.Parallel()

	opts := &dto.ReversalUpdateOptions{
		Source:         testutil.Ptr(models.SourceDirect),
		RelatedSteamID: testutil.Ptr(models.SteamID(76561197960287931)),
	}

	wantErr := "invalid related_steam_id and source combination"

	err := opts.Validate()
	if err == nil {
		t.Fatalf("Validate(): got nil error, wanted error")
	}
	if err.Error() != wantErr {
		t.Errorf("Validate(): got error %s, wanted %s", err.Error(), wantErr)
	}
}
