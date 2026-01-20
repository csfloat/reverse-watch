package dto

import (
	"testing"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/util"

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
				SteamID:         util.Ptr(models.SteamID(76561197960287930)),
				MarketplaceSlug: util.Ptr("test-slug"),
				Source:          util.Ptr(models.SourceRelatedUser),
				RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
				ReversedAt:      util.Ptr(uint64(1)),
				ExpungedAt:      util.Ptr(uint64(1)),
			},
			want: map[string]interface{}{
				"steam_id":         util.Ptr(models.SteamID(76561197960287930)),
				"marketplace_slug": util.Ptr("test-slug"),
				"source":           util.Ptr(models.SourceRelatedUser),
				"related_steam_id": util.Ptr(models.SteamID(76561197960287931)),
				"reversed_at":      util.Ptr(uint64(1)),
				"expunged_at":      util.Ptr(uint64(1)),
			},
		},
		{
			name: "sourceDirect",
			opts: &ReversalUpdateOptions{
				Source: util.Ptr(models.SourceDirect),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceDirect),
				"related_steam_id": nil,
			},
		},
		{
			name: "sourceRelatedUser",
			opts: &ReversalUpdateOptions{
				Source:         util.Ptr(models.SourceRelatedUser),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceRelatedUser),
				"related_steam_id": util.Ptr(models.SteamID(76561197960287931)),
			},
		},
		{
			name: "sourceUserReport",
			opts: &ReversalUpdateOptions{
				Source: util.Ptr(models.SourceUserReport),
			},
			want: map[string]interface{}{
				"source":           util.Ptr(models.SourceUserReport),
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
		SteamID:         util.Ptr(models.SteamID(76561197960287930)),
		MarketplaceSlug: util.Ptr("test-slug"),
		Source:          util.Ptr(models.SourceRelatedUser),
		RelatedSteamID:  util.Ptr(models.SteamID(76561197960287931)),
		ReversedAt:      util.Ptr(uint64(1)),
		ExpungedAt:      util.Ptr(uint64(1)),
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
				SteamID: util.Ptr(models.SteamID(0)),
			},
			wantErr: "steam_id is invalid",
		},
		{
			name: "emptyMarketplaceSlug",
			opts: &ReversalUpdateOptions{
				MarketplaceSlug: util.Ptr(""),
			},
			wantErr: "cannot set an empty marketplace_slug",
		},
		{
			name: "invalidSourceAndRelatedSteamID",
			opts: &ReversalUpdateOptions{
				Source:         util.Ptr(models.SourceDirect),
				RelatedSteamID: util.Ptr(models.SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "invalidReversedAt",
			opts: &ReversalUpdateOptions{
				ReversedAt: util.Ptr(uint64(0)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "reversedAtInFuture",
			opts: &ReversalUpdateOptions{
				ReversedAt: util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "reversed_at is invalid",
		},
		{
			name: "expungedAtInFuture",
			opts: &ReversalUpdateOptions{
				ExpungedAt: util.Ptr(uint64(99999999999999999)),
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
