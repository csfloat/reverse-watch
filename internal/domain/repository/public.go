package repository

import (
	"reverse-watch/internal/domain/models"
)

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *models.Cursor
	Limit           *uint
}

type ReversalUpdateOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Source          *models.Source
	RelatedSteamID  *models.SteamID
	ReversedAt      *uint64
	ExpungedAt      *uint64
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
	return fields
}

type ReversalRepository interface {
	Create(reversal *models.Reversal) error
	BulkCreate(reversals []*models.Reversal) error
	Read(id models.Snowflake) (*models.Reversal, error)
	Update(id models.Snowflake, opts *ReversalUpdateOptions) error
	Delete(id models.Snowflake) error
	Expunge(id models.Snowflake) error
	List(opts *ReversalListOptions) ([]*models.Reversal, error)
}

type PublicRepository interface {
	Reversal() ReversalRepository
}
