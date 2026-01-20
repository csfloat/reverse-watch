package models

import (
	"testing"

	"reverse-watch/internal/util"
)

func TestReversal_BeforeCreate(t *testing.T) {
	t.Parallel()

	InitSnowflakeGenerator(0 /* workerID */, 0 /* processID */)

	testReversal := &Reversal{
		SteamID:         SteamID(76561197960287930),
		MarketplaceSlug: "test-slug",
	}

	if err := testReversal.BeforeCreate(nil); err != nil {
		t.Fatalf("BeforeCreate(): %v", err)
	}
}

func TestReversal_BeforeCreate_Errors(t *testing.T) {
	t.Parallel()

	InitSnowflakeGenerator(0 /* workerID */, 0 /* processID */)

	testCases := []struct {
		name     string
		reversal *Reversal
		wantErr  string
	}{
		{
			name: "invalidSteamID",
			reversal: &Reversal{
				SteamID: 0,
			},
			wantErr: "steam_id is invalid",
		},
		{
			name: "emptyMarketplaceSlug",
			reversal: &Reversal{
				SteamID:         SteamID(76561197960287930),
				MarketplaceSlug: "",
			},
			wantErr: "marketplace_slug is required",
		},
		{
			name: "invalidRelatedSteamIDAndSource",
			reversal: &Reversal{
				SteamID:         SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				Source:          util.Ptr(SourceDirect),
				RelatedSteamID:  util.Ptr(SteamID(76561197960287931)),
			},
			wantErr: "invalid related_steam_id and source combination",
		},
		{
			name: "reversedAtInFuture",
			reversal: &Reversal{
				SteamID:         SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ReversedAt:      uint64(99999999999999999),
			},
			wantErr: "reversed_at cannot be in the future",
		},
		{
			name: "expungedAtInFuture",
			reversal: &Reversal{
				SteamID:         SteamID(76561197960287930),
				MarketplaceSlug: "test-slug",
				ExpungedAt:      util.Ptr(uint64(99999999999999999)),
			},
			wantErr: "expunged_at cannot be in the future",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.reversal.BeforeCreate(nil)
			if err == nil {
				t.Fatalf("got nil error, wanted %v", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("got %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}
