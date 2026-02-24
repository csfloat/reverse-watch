package dto

import (
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/errors"
)

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *Cursor
	Limit           *uint
	OrderParam      *OrderParam
}

type ReversalUpdates struct {
	Source         *models.Source  `json:"source"`
	RelatedSteamID *models.SteamID `json:"related_steam_id"`
	ReversedAt     *uint64         `json:"reversed_at"`
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

	// Ensure related_steam_id is nullified when source is not SourceRelatedUser
	if u.Source != nil && *u.Source != models.SourceRelatedUser {
		fields["related_steam_id"] = nil
	}
	return fields
}

func (u *ReversalUpdates) Validate() error {
	if u == nil {
		return errors.New(errors.BadRequest, "reversal updates cannot be nil")
	}
	if len(u.ToFields()) == 0 {
		return errors.New(errors.BadRequest, "reversal updates must have at least one field")
	}

	if err := models.ValidateSourceAndRelatedID(u.Source, u.RelatedSteamID); err != nil {
		return errors.New(errors.BadRequest, err.Error())
	}

	now := uint64(time.Now().UnixMilli())
	if u.ReversedAt != nil {
		if *u.ReversedAt == 0 || *u.ReversedAt < models.Epoch || *u.ReversedAt > now {
			return errors.New(errors.BadRequest, "reversed_at is invalid")
		}
	}
	return nil
}
