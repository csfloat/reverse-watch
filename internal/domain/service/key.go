package service

import "reverse-watch/internal/domain/models"

type KeyService interface {
	CreateKey(marketplaceSlug string, permissions models.Permissions) (*models.RawKey, error)
	GetKey(id models.Snowflake) (*models.Key, error)
	UpdateKey(id models.Snowflake, opts *UpdateKeyOptions) error
	DeleteKey(id models.Snowflake) error
	ListKeys(opts *ListKeyOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}

type UpdateKeyOptions struct {
	Permissions *models.Permissions
}

type ListKeyOptions struct {
	MarketplaceSlug string
}
