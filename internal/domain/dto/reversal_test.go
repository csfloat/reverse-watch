package dto

import (
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/testutil"

	"github.com/google/go-cmp/cmp"
)

func TestReversalUpdateOptions_ToFields(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		opts *ReversalUpdateOptions
		want map[string]interface{}
	}{
		{
			name: "allFields",
			opts: &ReversalUpdateOptions{
				SteamID:         testutil.Ptr(models.SteamID(76561197960287930)),
				MarketplaceSlug: testutil.Ptr("test-slug"),
				Source:          testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
				ReversedAt:      testutil.Ptr(uint64(1)),
				ExpungedAt:      testutil.Ptr(uint64(1)),
			},
			want: map[string]interface{}{
				"steam_id":         testutil.Ptr(models.SteamID(76561197960287930)),
				"marketplace_slug": testutil.Ptr("test-slug"),
				"source":           testutil.Ptr(models.SourceRelatedUser),
				"related_steam_id": testutil.Ptr(models.SteamID(76561197960287931)),
				"reversed_at":      testutil.Ptr(uint64(1)),
				"expunged_at":      testutil.Ptr(uint64(1)),
			},
		},
		{
			name: "sourceDirect",
			opts: &ReversalUpdateOptions{
				Source: testutil.Ptr(models.SourceDirect),
			},
			want: map[string]interface{}{
				"source":           testutil.Ptr(models.SourceDirect),
				"related_steam_id": nil,
			},
		},
		{
			name: "sourceRelatedUser",
			opts: &ReversalUpdateOptions{
				Source:         testutil.Ptr(models.SourceRelatedUser),
				RelatedSteamID: testutil.Ptr(models.SteamID(76561197960287931)),
			},
			want: map[string]interface{}{
				"source":           testutil.Ptr(models.SourceRelatedUser),
				"related_steam_id": testutil.Ptr(models.SteamID(76561197960287931)),
			},
		},
		{
			name: "sourceUserReport",
			opts: &ReversalUpdateOptions{
				Source: testutil.Ptr(models.SourceUserReport),
			},
			want: map[string]interface{}{
				"source":           testutil.Ptr(models.SourceUserReport),
				"related_steam_id": nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.opts.ToFields(), tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReversalUpdateOptions_Validate(t *testing.T) {
	t.Parallel()

	opts := &ReversalUpdateOptions{
		SteamID:         testutil.Ptr(models.SteamID(76561197960287930)),
		MarketplaceSlug: testutil.Ptr("test-slug"),
		Source:          testutil.Ptr(models.SourceRelatedUser),
		RelatedSteamID:  testutil.Ptr(models.SteamID(76561197960287931)),
		ReversedAt:      testutil.Ptr(uint64(1)),
		ExpungedAt:      testutil.Ptr(uint64(1)),
	}

	if err := opts.Validate(); err != nil {
		t.Fatalf("Validate(): %v", err)
	}
}

func TestReversalUpdateOptions_Validate_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		opts    *ReversalUpdateOptions
		wantErr string
	}{
		{
			name: "invalidSteamID",
			opts: &ReversalUpdateOptions{
				SteamID: testutil.Ptr(models.SteamID(0)),
			},
			wantErr: "steam_id is invalid",
		},
		{
			name: "emptyMarketplaceSlug",
			opts: &ReversalUpdateOptions{
				MarketplaceSlug: testutil.Ptr(""),
			},
			wantErr: "cannot set an empty marketplace_slug",
		},
		{
			name: "invalidSourceAndRelatedSteamID",
			opts: &ReversalUpdateOptions{
				Source:         testutil.Ptr(models.SourceDirect),
				RelatedSteamID: testutil.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "reversedAtInFuture",
			opts: &ReversalUpdateOptions{
				ReversedAt: testutil.Ptr(uint64(99999999999999999)),
			},
			wantErr: "reversed_at cannot be in the future",
		},
		{
			name: "expungedAtInFuture",
			opts: &ReversalUpdateOptions{
				ExpungedAt: testutil.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at cannot be in the future",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.opts.Validate()
			if err == nil {
				t.Fatalf("Validate(): got nil error, wanted %s", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("Validate(): got error %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}
