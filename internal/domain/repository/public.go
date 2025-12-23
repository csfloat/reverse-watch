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

type ReversalRepository interface {
	Create(reversal *models.Reversal) error
	BulkCreate(reversals []*models.Reversal) error
	Read(id models.Snowflake) (*models.Reversal, error)
	Update(id models.Snowflake, fields map[string]interface{}) error
	Delete(id models.Snowflake) error
	Expunge(id models.Snowflake) error
	List(opts *ReversalListOptions) ([]*models.Reversal, error)
}

type PublicRepository interface {
	Reversal() ReversalRepository
}
