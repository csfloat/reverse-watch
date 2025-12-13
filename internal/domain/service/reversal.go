package service

import "reverse-watch/internal/domain/models"

type ReversalService interface {
	CreateReversal(reversal *models.Reversal) error
	GetReversal(id models.Snowflake) (*models.Reversal, error)
	UpdateReversal(id models.Snowflake, fields map[string]interface{}) error
	DeleteReversal(id models.Snowflake) error
	ListReversals(opts *ListReversalOptions) ([]*models.Reversal, error)
}

type ListReversalOptions struct {
	SteamID         models.SteamID
	MarketplaceSlug string
	Cursor          string
	Limit           uint
}
