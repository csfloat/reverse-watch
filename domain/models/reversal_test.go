package models

import (
	"testing"

	"reverse-watch/util"
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

func TestValidateSourceAndRelatedID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		source         *Source
		relatedSteamID *SteamID
	}{
		{
			name:           "bothNil",
			source:         nil,
			relatedSteamID: nil,
		},
		{
			name:           "validSource",
			source:         util.Ptr(SourceDirect),
			relatedSteamID: nil,
		},
		{
			name:           "validSourceAndRelatedSteamID",
			source:         util.Ptr(SourceRelatedUser),
			relatedSteamID: util.Ptr(SteamID(76561197960287930)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateSourceAndRelatedID(tc.source, tc.relatedSteamID); err != nil {
				t.Fatalf("got error %v, wanted nil", err)
			}
		})
	}
}

func TestValidateSourceAndRelatedID_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		source         *Source
		relatedSteamID *SteamID
		wantErr        string
	}{
		{
			name:           "invalidRelatedSteamID",
			relatedSteamID: util.Ptr(SteamID(0)),
			wantErr:        "related_steam_id is invalid",
		},
		{
			name:           "validRelatedSteamIDWithNilSource",
			source:         nil,
			relatedSteamID: util.Ptr(SteamID(76561197960287931)),
			wantErr:        "invalid related_steam_id and source combination",
		},
		{
			name:           "invalidSourceAndRelatedSteamID",
			source:         util.Ptr(SourceDirect),
			relatedSteamID: util.Ptr(SteamID(76561197960287931)),
			wantErr:        "invalid related_steam_id and source combination",
		},
		{
			name:           "validSourceWithNilRelatedSteamID",
			source:         util.Ptr(SourceRelatedUser),
			relatedSteamID: nil,
			wantErr:        "related_steam_id is required when source is \"related_user\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSourceAndRelatedID(tc.source, tc.relatedSteamID)
			if err == nil {
				t.Fatalf("got nil error, wanted %v", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Fatalf("got %v, wanted %v", err, tc.wantErr)
			}
		})
	}
}
