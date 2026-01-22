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
	Source         *models.Source  `json:"source"`
	RelatedSteamID *models.SteamID `json:"related_steam_id"`
	ReversedAt     *uint64         `json:"reversed_at"`
	ExpungedAt     *uint64         `json:"expunged_at"`
}

func (o *ReversalUpdateOptions) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
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
	if err := models.ValidateSourceAndRelatedID(o.Source, o.RelatedSteamID); err != nil {
		return err
	}

	midnightJan012025 := uint64(1735714800000)
	now := uint64(time.Now().UnixMilli())
	if o.ReversedAt != nil {
		if *o.ReversedAt == 0 || *o.ReversedAt < midnightJan012025 || *o.ReversedAt > now {
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
