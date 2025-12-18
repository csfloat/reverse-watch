package service

import "reverse-watch/internal/domain/models"

type ReversalService interface {
	CreateReversal(reversal *models.Reversal) error
	BulkCreateReversals(reversals []*models.Reversal) error
	GetReversal(id models.Snowflake) (*models.Reversal, error)
	UpdateReversal(id models.Snowflake, fields map[string]interface{}) error
	DeleteReversal(id models.Snowflake) error
	ExpungeReversal(id models.Snowflake) error
	ListReversals(opts *ReversalListOptions) ([]*models.Reversal, error)
}

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *models.Cursor
	Limit           *uint
}
