package dto

import (
	"fmt"
	"time"

	"reverse-watch/domain/models"
)

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *Cursor
	Limit           *uint
}

type ReversalUpdates struct {
	Source         *models.Source  `json:"source"`
	RelatedSteamID *models.SteamID `json:"related_steam_id"`
	ReversedAt     *uint64         `json:"reversed_at"`
	ExpungedAt     *uint64         `json:"expunged_at"`
}

func (u *ReversalUpdates) ToFields() map[string]interface{} {
	fields := make(map[string]interface{})
	if u.Source != nil {
		fields["source"] = u.Source
	}
	if u.RelatedSteamID != nil {
		fields["related_steam_id"] = u.RelatedSteamID
	}
	if u.ReversedAt != nil {
		fields["reversed_at"] = u.ReversedAt
	}
	if u.ExpungedAt != nil {
		fields["expunged_at"] = u.ExpungedAt
	}

	// Ensure related_steam_id is nullified when source is not SourceRelatedUser
	if u.Source != nil && *u.Source != models.SourceRelatedUser {
		fields["related_steam_id"] = nil
	}
	return fields
}

func (u *ReversalUpdates) Validate() error {
	if u == nil {
		return fmt.Errorf("reversal updates cannot be nil")
	}
	if len(u.ToFields()) == 0 {
		return fmt.Errorf("reversal updates must have at least one field")
	}

	if err := models.ValidateSourceAndRelatedID(u.Source, u.RelatedSteamID); err != nil {
		return err
	}

	now := uint64(time.Now().UnixMilli())
	if u.ReversedAt != nil {
		if *u.ReversedAt == 0 || *u.ReversedAt < models.Epoch || *u.ReversedAt > now {
			return fmt.Errorf("reversed_at is invalid")
		}
	}

	if u.ExpungedAt != nil {
		if *u.ExpungedAt == 0 || *u.ExpungedAt < models.Epoch || *u.ExpungedAt > now {
			return fmt.Errorf("expunged_at is invalid")
		}
	}
	return nil
}
