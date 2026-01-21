package dto

import (
	"fmt"
	"time"

	"reverse-watch/internal/domain/models"
)

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *Cursor
	Limit           *uint
}

type ReversalUpdateOptions struct {
	SteamID         *models.SteamID `json:"steam_id"`
	MarketplaceSlug *string         `json:"marketplace_slug"`
	Source          *models.Source  `json:"source"`
	RelatedSteamID  *models.SteamID `json:"related_steam_id"`
	ReversedAt      *uint64         `json:"reversed_at"`
	ExpungedAt      *uint64         `json:"expunged_at"`
}

func (o *ReversalUpdateOptions) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if o.SteamID != nil {
		fields["steam_id"] = o.SteamID
	}
	if o.MarketplaceSlug != nil {
		fields["marketplace_slug"] = o.MarketplaceSlug
	}
	if o.Source != nil {
		fields["source"] = o.Source
	}
	if o.RelatedSteamID != nil {
		fields["related_steam_id"] = o.RelatedSteamID
	}
	if o.ReversedAt != nil {
		fields["reversed_at"] = o.ReversedAt
	}
	if o.ExpungedAt != nil {
		fields["expunged_at"] = o.ExpungedAt
	}

	// Ensure related_steam_id is nullified when source is not SourceRelatedUser
	if o.Source != nil && *o.Source != models.SourceRelatedUser {
		fields["related_steam_id"] = nil
	}
	return fields
}

func (o *ReversalUpdateOptions) Validate() error {
	if o.SteamID != nil {
		if !o.SteamID.IsValid() {
			return fmt.Errorf("steam_id is invalid")
		}
	}
	if o.MarketplaceSlug != nil {
		if *o.MarketplaceSlug == "" {
			return fmt.Errorf("cannot set an empty marketplace_slug")
		}
	}

	if err := models.ValidateSourceAndRelatedID(o.Source, o.RelatedSteamID); err != nil {
		return err
	}

	now := uint64(time.Now().UnixMilli())
	if o.ReversedAt != nil {
		if *o.ReversedAt == 0 || *o.ReversedAt > now {
			return fmt.Errorf("reversed_at is invalid")
		}
	}

	if o.ExpungedAt != nil {
		if *o.ExpungedAt == 0 || *o.ExpungedAt > now {
			return fmt.Errorf("expunged_at is invalid")
		}
	}
	return nil
}
