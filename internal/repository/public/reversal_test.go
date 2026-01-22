package public

import (
	"errors"
	"sort"
	"testing"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"
	"reverse-watch/internal/util"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gorm.io/gorm"
)

func TestReversalRepository_BeforeCreate(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
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

	db := testutil.NewTestDB(t)
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
				RelatedSteamID:  util.Ptr(models.SteamID(0)),
			},
			wantErr: "related_steam_id is invalid",
		},
		{
			name: "validRelatedSteamIDWithInvalidSource",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          util.Ptr(models.SourceDirect),
				RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "validSourceWithNilRelatedSteamID",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          util.Ptr(models.SourceRelatedUser),
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
				ExpungedAt:      util.Ptr(uint64(0)),
			},
			wantErr: "expunged_at is invalid",
		},
		{
			name: "expungedAtInFuture",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ExpungedAt:      util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at is invalid",
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

	db := testutil.NewTestDB(t)
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
					Source:          util.Ptr(models.SourceRelatedUser),
					RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
				},
			},
		},
		{
			name: "sourceDirect",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
					Source:          util.Ptr(models.SourceDirect),
				},
			},
		},
		{
			name: "sourceUserReport",
			reversals: []*models.Reversal{
				{
					SteamID:         models.SteamID(76561197960287930),
					MarketplaceSlug: "test-slug-1",
					Source:          util.Ptr(models.SourceUserReport),
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

	db := testutil.NewTestDB(t)
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

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
		Source:          util.Ptr(models.SourceRelatedUser),
		RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
		ExpungedAt:      util.Ptr(models.Epoch + 1),
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

	testCases := []struct {
		name         string
		initial      *models.Reversal
		opts         *dto.ReversalUpdateOptions
		want         *models.Reversal
		ignoreFields []string
	}{
		{
			name: "source",
			initial: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
			},
			opts: &dto.ReversalUpdateOptions{
				Source: util.Ptr(models.SourceDirect),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          util.Ptr(models.SourceDirect),
			},
			ignoreFields: []string{"CreatedAt", "UpdatedAt", "ReversedAt"},
		},
		{
			name: "relatedUser",
			initial: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
			},
			opts: &dto.ReversalUpdateOptions{
				Source:         util.Ptr(models.SourceRelatedUser),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          util.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
			},
			ignoreFields: []string{"CreatedAt", "UpdatedAt", "ReversedAt"},
		},
		{
			name: "reversedAt",
			initial: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
			},
			opts: &dto.ReversalUpdateOptions{
				ReversedAt: util.Ptr(models.Epoch + 1),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ReversedAt:      models.Epoch + 1,
			},
			ignoreFields: []string{"CreatedAt", "UpdatedAt"},
		},
		{
			name: "expungedAt",
			initial: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
			},
			opts: &dto.ReversalUpdateOptions{
				ExpungedAt: util.Ptr(models.Epoch + 1),
			},
			want: &models.Reversal{
				Model: models.Model{
					ID: models.Snowflake(1),
				},
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ExpungedAt:      util.Ptr(models.Epoch + 1),
			},
			ignoreFields: []string{"CreatedAt", "UpdatedAt", "ReversedAt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := testutil.NewTestDB(t)
			reversalRepo := NewReversalRepository(db)

			testutil.Insert(t, db, tc.initial)

			if err := reversalRepo.Update(tc.initial.ID, tc.opts); err != nil {
				t.Fatalf("Update(): %v", err)
			}

			var got models.Reversal
			if err := db.Where("id = ?", tc.initial.ID).First(&got).Error; err != nil {
				t.Fatalf("First(): %v", err)
			}

			if diff := cmp.Diff(tc.want, &got, cmpopts.IgnoreFields(models.Reversal{}, tc.ignoreFields...)); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalRepository_Update_Errors(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
	}
	testutil.Insert(t, db, testReversal)

	testCases := []struct {
		name    string
		id      models.Snowflake
		opts    *dto.ReversalUpdateOptions
		wantErr string
	}{
		{
			name: "notFound",
			id:   models.Snowflake(0),
			opts: &dto.ReversalUpdateOptions{
				Source: util.Ptr(models.SourceDirect),
			},
			wantErr: gorm.ErrRecordNotFound.Error(),
		},
		{
			name:    "nilOptions",
			id:      models.Snowflake(1),
			opts:    nil,
			wantErr: "opts cannot be nil",
		},
		{
			name:    "emptyOptions",
			id:      models.Snowflake(1),
			opts:    &dto.ReversalUpdateOptions{},
			wantErr: "no fields to update",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reversalRepo.Update(tc.id, tc.opts)
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
		Source:         util.Ptr(models.SourceDirect),
		RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
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

func TestReversalRepository_Delete(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
	}
	testutil.Insert(t, db, testReversal)

	if err := reversalRepo.Delete(testReversal.ID); err != nil {
		t.Fatalf("Delete(): %v", err)
	}

	var deletedReversal models.Reversal
	err := db.Where("id = ?", testReversal.ID).First(&deletedReversal).Error
	if err == nil {
		t.Fatalf("First(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("First(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestReversalRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	err := reversalRepo.Delete(models.Snowflake(1))
	if err == nil {
		t.Fatalf("Delete(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Delete(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestReversalRepository_List(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversals := []*models.Reversal{
		{
			Model: models.Model{
				ID:        1,
				CreatedAt: 1,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID:        2,
				CreatedAt: 2,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "another-test-slug",
		},
		{
			Model: models.Model{
				ID:        3,
				CreatedAt: 3,
			},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "test-slug",
		},
	}
	testutil.Insert(t, db, testReversals...)

	testCases := []struct {
		name string
		opts *dto.ReversalListOptions
		want []*models.Reversal
	}{
		{
			name: "nilOptions",
			opts: nil,
			want: testReversals,
		},
		{
			name: "emptyOptions",
			opts: &dto.ReversalListOptions{},
			want: testReversals,
		},
		{
			name: "bySteamID",
			opts: &dto.ReversalListOptions{
				SteamID: util.Ptr(models.SteamID(76561197960287930)),
			},
			want: []*models.Reversal{
				testReversals[0],
				testReversals[1],
			},
		},
		{
			name: "byMarketplaceSlug",
			opts: &dto.ReversalListOptions{
				MarketplaceSlug: util.Ptr("test-slug"),
			},
			want: []*models.Reversal{
				testReversals[0],
				testReversals[2],
			},
		},
		{
			name: "withCursor",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID:        3,
					CreatedAt: 3,
				},
			},
			want: []*models.Reversal{
				testReversals[0],
				testReversals[1],
			},
		},
		{
			name: "withLimit",
			opts: &dto.ReversalListOptions{
				Limit: util.Ptr(uint(1)),
			},
			want: []*models.Reversal{
				testReversals[2],
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reversalRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}

			sort.Slice(got, func(i, j int) bool {
				return got[i].ID < got[j].ID
			})

			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalRepository_List_Pagination(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversals := []*models.Reversal{
		{
			Model: models.Model{
				ID:        50,
				CreatedAt: 100,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID:        40,
				CreatedAt: 100,
			},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID:        30,
				CreatedAt: 100,
			},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "another-test-slug",
		},
		{
			Model: models.Model{
				ID:        60,
				CreatedAt: 90,
			},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID:        55,
				CreatedAt: 90,
			},
			SteamID:         models.SteamID(76561197960287934),
			MarketplaceSlug: "test-slug",
		},
	}
	testutil.Insert(t, db, testReversals...)

	testCases := []struct {
		name   string
		cursor *dto.Cursor
		limit  uint
		want   []*models.Reversal
	}{
		{
			name:  "firstPage",
			limit: 2,
			want: []*models.Reversal{
				testReversals[0],
				testReversals[1],
			},
		},
		{
			name: "secondPage",
			cursor: &dto.Cursor{
				ID:        40,
				CreatedAt: 100,
			},
			limit: 2,
			want: []*models.Reversal{
				testReversals[2],
				testReversals[3],
			},
		},
		{
			name: "thirdPage",
			cursor: &dto.Cursor{
				ID:        60,
				CreatedAt: 90,
			},
			limit: 2,
			want: []*models.Reversal{
				testReversals[4],
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reversalRepo.List(&dto.ReversalListOptions{
				Cursor: tc.cursor,
				Limit:  &tc.limit,
			})
			if err != nil {
				t.Fatalf("List(): %v", err)
			}
			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}
