package public

import (
	"errors"
	"testing"
	"time"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	rwerrors "reverse-watch/errors"
	"reverse-watch/internal/testutil"
	"reverse-watch/util"

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
		name     string
		reversal *models.Reversal
	}{
		{
			name: "singleReversal",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug-1",
			},
		},
		{
			name: "validSourceAndRelatedSteamID",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug-1",
				Source:          util.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
			},
		},
		{
			name: "sourceDirect",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug-1",
				Source:          util.Ptr(models.SourceDirect),
			},
		},
		{
			name: "sourceUserReport",
			reversal: &models.Reversal{
				SteamID:         models.SteamID(76561197960287930),
				MarketplaceSlug: "test-slug-1",
				Source:          util.Ptr(models.SourceUserReport),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := reversalRepo.Create(tc.reversal); err != nil {
				t.Fatalf("Create(): %v", err)
			}

			var storedReversal models.Reversal
			if err := db.Where("id = ?", tc.reversal.ID).Find(&storedReversal).Error; err != nil {
				t.Fatalf("failed to retrieve stored reversals: %v", err)
			}

			if diff := cmp.Diff(storedReversal, *tc.reversal, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt")); diff != "" {
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

func TestReversalRepository_BulkCreate(t *testing.T) {
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
			if err := reversalRepo.BulkCreate(tc.reversals); err != nil {
				t.Fatalf("BulkCreate(): %v", err)
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

func TestReversalRepository_BulkCreate_Errors(t *testing.T) {
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
			err := reversalRepo.BulkCreate([]*models.Reversal{tc.reversal})
			if err == nil {
				t.Fatalf("BulkCreate(): got nil error, wanted error")
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("BulkCreate(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}

func TestReversalRepository_Read(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversal := &models.Reversal{
		Model: models.Model{
			ID:        1,
			CreatedAt: models.Epoch + 1,
			UpdatedAt: models.Epoch + 1,
		},
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
		Source:          util.Ptr(models.SourceRelatedUser),
		RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
		ExpungedAt:      util.Ptr(uint64(time.Now().Add(-24 * time.Hour).UnixMilli())),
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
		updates      *dto.ReversalUpdates
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
			updates: &dto.ReversalUpdates{
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
			updates: &dto.ReversalUpdates{
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
			updates: &dto.ReversalUpdates{
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
			updates: &dto.ReversalUpdates{
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

			if err := reversalRepo.Update(tc.initial.ID, tc.updates); err != nil {
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
		updates *dto.ReversalUpdates
		wantErr string
	}{
		{
			name: "notFound",
			id:   models.Snowflake(0),
			updates: &dto.ReversalUpdates{
				Source: util.Ptr(models.SourceDirect),
			},
			wantErr: gorm.ErrRecordNotFound.Error(),
		},
		{
			name:    "nilUpdates",
			id:      models.Snowflake(1),
			updates: nil,
			wantErr: rwerrors.New(rwerrors.BadRequest, "reversal updates cannot be nil").Error(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := reversalRepo.Update(tc.id, tc.updates)
			if err == nil {
				t.Fatalf("Update(): got nil error, wanted error")
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Update(): got error %s, wanted %s", err.Error(), tc.wantErr)
			}
		})
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
	// Ensure db doesn't return the deleted record
	err := db.Where("id = ?", testReversal.ID).First(&deletedReversal).Error
	if err == nil {
		t.Fatalf("First(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("First(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}

	// Ensure the reversal was soft deleted
	err = db.Unscoped().Where("id = ?", testReversal.ID).First(&deletedReversal).Error
	if err != nil {
		t.Fatalf("Unscoped().First(): %v", err)
	}
	if deletedReversal.DeletedAt.Time.IsZero() {
		t.Fatalf("DeletedAt is zero")
	}
	if diff := cmp.Diff(testReversal, &deletedReversal, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "DeletedAt", "ReversedAt")); diff != "" {
		t.Fatal(diff)
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

func TestReversalRepository_DeleteUser(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversals := []*models.Reversal{
		{
			Model:           models.Model{ID: 1},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "test-slug",
		},
		{
			Model:           models.Model{ID: 2},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "test-slug",
		},
		{
			Model:           models.Model{ID: 3},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "another-test-slug",
		},
	}
	testutil.Insert(t, db, testReversals...)

	if err := reversalRepo.DeleteAllUserReports(testReversals[0].SteamID); err != nil {
		t.Fatalf("DeleteAllUserReports(): %v", err)
	}

	var deletedReversals []*models.Reversal
	if err := db.Unscoped().Where("steam_id = ?", testReversals[0].SteamID).Find(&deletedReversals).Error; err != nil {
		t.Fatalf("Find(): %v", err)
	}

	if len(deletedReversals) > 0 {
		t.Fatalf("wanted 0 reversals, got %d", len(deletedReversals))
	}
}

func TestReversalRepository_DeleteUser_NotFound(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	err := reversalRepo.DeleteAllUserReports(models.SteamID(1))
	if err == nil {
		t.Fatalf("DeleteAllUserReports(): got nil error, wanted error")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("DeleteAllUserReports(): got error %v, wanted %v", err, gorm.ErrRecordNotFound)
	}
}

func TestReversalRepository_List(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	testReversals := []*models.Reversal{
		{
			Model: models.Model{
				ID: 1,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID: 2,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "another-test-slug",
		},
		{
			Model: models.Model{
				ID: 3,
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
			name: "withCursorDesc",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 3,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
			},
			want: []*models.Reversal{
				testReversals[1],
				testReversals[0],
			},
		},
		{
			name: "withCursorAsc",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 1,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.ASC,
				},
			},
			want: []*models.Reversal{
				testReversals[1],
				testReversals[2],
			},
		},
		{
			name: "withLimit",
			opts: &dto.ReversalListOptions{
				Limit: util.Ptr(uint(1)),
			},
			want: []*models.Reversal{
				testReversals[0],
			},
		},
		{
			name: "withOrder",
			opts: &dto.ReversalListOptions{
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
			},
			want: []*models.Reversal{
				testReversals[2],
				testReversals[1],
				testReversals[0],
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reversalRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}

			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalRepository_SummaryStats(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	now := uint64(time.Now().UnixMilli())
	hourMs := uint64(60 * 60 * 1000)
	withinDay := now - hourMs       // -1h: counts as "last 24h"
	olderThanDay := now - 36*hourMs // -36h: outside "last 24h"

	// A: 1 non-expunged within 24h + 1 non-expunged older  -> indexed, flagged, flagged_24h
	// B: 1 expunged within 24h + 1 non-expunged older      -> indexed, flagged (not 24h)
	// C: 1 expunged older                                  -> indexed only
	// D: 1 non-expunged within 24h                         -> indexed, flagged, flagged_24h
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: withinDay},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat-2",
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: withinDay},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(now),
		},
		&models.Reversal{
			Model:           models.Model{ID: 4, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat-2",
		},
		&models.Reversal{
			Model:           models.Model{ID: 5, CreatedAt: olderThanDay},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(now - 24*hourMs),
		},
		&models.Reversal{
			Model:           models.Model{ID: 6, CreatedAt: withinDay},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "csfloat",
		},
	)

	got, err := reversalRepo.SummaryStats()
	if err != nil {
		t.Fatalf("SummaryStats(): %v", err)
	}
	want := &dto.SummaryStats{
		TradersIndexed:    4,
		TradersFlagged:    3,
		TradersFlagged24h: 2,
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SummaryStats() mismatch (-want +got):\n%s", diff)
	}
}

func TestReversalRepository_DailyCounts(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayMs := func(offset int, addHours int) uint64 {
		return uint64(today.AddDate(0, 0, offset).Add(time.Duration(addHours) * time.Hour).UnixMilli())
	}

	// today:    2 non-expunged (very early today, safely past)
	// today-1:  1 non-expunged + 1 expunged (excluded)
	// today-2:  1 non-expunged
	// today-3:  1 non-expunged (outside days=3 window)
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
			ReversedAt:      uint64(today.UnixMilli()) + 1,
		},
		&models.Reversal{
			Model:           models.Model{ID: 2},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
			ReversedAt:      uint64(today.UnixMilli()) + 2,
		},
		&models.Reversal{
			Model:           models.Model{ID: 3},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ReversedAt:      dayMs(-1, 12),
		},
		&models.Reversal{
			Model:           models.Model{ID: 4, CreatedAt: dayMs(-1, 15)},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "csfloat",
			ReversedAt:      dayMs(-1, 15),
			ExpungedAt:      util.Ptr(dayMs(-1, 16)),
		},
		&models.Reversal{
			Model:           models.Model{ID: 5},
			SteamID:         models.SteamID(76561197960287934),
			MarketplaceSlug: "csfloat",
			ReversedAt:      dayMs(-2, 5),
		},
		&models.Reversal{
			Model:           models.Model{ID: 6},
			SteamID:         models.SteamID(76561197960287935),
			MarketplaceSlug: "csfloat",
			ReversedAt:      dayMs(-3, 1),
		},
	)

	got, err := reversalRepo.DailyCounts(3)
	if err != nil {
		t.Fatalf("DailyCounts(3): %v", err)
	}
	want := []dto.DailyCount{
		{Date: today.AddDate(0, 0, -2).Format("2006-01-02"), Count: 1},
		{Date: today.AddDate(0, 0, -1).Format("2006-01-02"), Count: 1},
		{Date: today.Format("2006-01-02"), Count: 2},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("DailyCounts(3) mismatch (-want +got):\n%s", diff)
	}
}

func TestReversalRepository_DailyCounts_ZeroFill(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	got, err := reversalRepo.DailyCounts(5)
	if err != nil {
		t.Fatalf("DailyCounts(5): %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("DailyCounts(5): got %d buckets, want 5", len(got))
	}
	for i, b := range got {
		wantDate := today.AddDate(0, 0, -(4 - i)).Format("2006-01-02")
		if b.Date != wantDate {
			t.Errorf("bucket[%d].Date = %q, want %q", i, b.Date, wantDate)
		}
		if b.Count != 0 {
			t.Errorf("bucket[%d].Count = %d, want 0", i, b.Count)
		}
	}
}

func TestReversalRepository_List_ExcludeExpunged(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	reversalRepo := NewReversalRepository(db)

	base := models.Epoch + 1000

	// 5 rows with strictly increasing CreatedAt. Row id=3 is expunged.
	testutil.Insert(t, db,
		&models.Reversal{
			Model:           models.Model{ID: 1, CreatedAt: base + 100},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 2, CreatedAt: base + 200},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 3, CreatedAt: base + 300},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "csfloat",
			ExpungedAt:      util.Ptr(base + 400),
		},
		&models.Reversal{
			Model:           models.Model{ID: 4, CreatedAt: base + 500},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "csfloat",
		},
		&models.Reversal{
			Model:           models.Model{ID: 5, CreatedAt: base + 600},
			SteamID:         models.SteamID(76561197960287934),
			MarketplaceSlug: "csfloat",
		},
	)

	testCases := []struct {
		name    string
		opts    *dto.ReversalListOptions
		wantIDs []models.Snowflake
	}{
		{
			name: "newestFirstExcludingExpunged",
			opts: &dto.ReversalListOptions{
				ExcludeExpunged: true,
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
			},
			wantIDs: []models.Snowflake{5, 4, 2, 1},
		},
		{
			name: "respectsLimit",
			opts: &dto.ReversalListOptions{
				ExcludeExpunged: true,
				Limit:           util.Ptr[uint](2),
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
			},
			wantIDs: []models.Snowflake{5, 4},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reversalRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("List(): got %d rows, want %d", len(got), len(tc.wantIDs))
			}
			for i, wantID := range tc.wantIDs {
				if got[i].ID != wantID {
					t.Errorf("List()[%d].ID = %d, want %d", i, got[i].ID, wantID)
				}
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
				ID: 60,
			},
			SteamID:         models.SteamID(76561197960287930),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID: 50,
			},
			SteamID:         models.SteamID(76561197960287931),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID: 40,
			},
			SteamID:         models.SteamID(76561197960287932),
			MarketplaceSlug: "another-test-slug",
		},
		{
			Model: models.Model{
				ID: 30,
			},
			SteamID:         models.SteamID(76561197960287933),
			MarketplaceSlug: "test-slug",
		},
		{
			Model: models.Model{
				ID: 20,
			},
			SteamID:         models.SteamID(76561197960287934),
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
			name: "firstPageDESC",
			opts: &dto.ReversalListOptions{
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
				Limit: util.Ptr[uint](2),
			},
			want: []*models.Reversal{
				testReversals[0],
				testReversals[1],
			},
		},
		{
			name: "secondPageDESC",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 50,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
				Limit: util.Ptr[uint](2),
			},
			want: []*models.Reversal{
				testReversals[2],
				testReversals[3],
			},
		},
		{
			name: "thirdPageDESC",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 30,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.DESC,
				},
				Limit: util.Ptr[uint](2),
			},
			want: []*models.Reversal{
				testReversals[4],
			},
		},
		{
			name: "firstPageASC",
			opts: &dto.ReversalListOptions{
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.ASC,
				},
				Limit: util.Ptr[uint](2),
			},
			want: []*models.Reversal{
				testReversals[4],
				testReversals[3],
			},
		},
		{
			name: "secondPageASC",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 30,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.ASC,
				},
				Limit: util.Ptr[uint](2),
			},
			want: []*models.Reversal{
				testReversals[2],
				testReversals[1],
			},
		},
		{
			name: "thirdPageASC",
			opts: &dto.ReversalListOptions{
				Cursor: &dto.Cursor{
					ID: 50,
				},
				OrderParam: &dto.OrderParam{
					Column:    "id",
					Direction: dto.ASC,
				},
			},
			want: []*models.Reversal{
				testReversals[0],
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := reversalRepo.List(tc.opts)
			if err != nil {
				t.Fatalf("List(): %v", err)
			}
			if diff := cmp.Diff(got, tc.want, cmpopts.IgnoreFields(models.Reversal{}, "CreatedAt", "UpdatedAt", "ReversedAt")); diff != "" {
				t.Error(diff)
			}
		})
	}
}
