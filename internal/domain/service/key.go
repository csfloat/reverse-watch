package service

import "reverse-watch/internal/domain/models"

type KeyService interface {
	CreateKey(marketplaceSlug string, scope models.Scope) (*models.RawKey, error)
	GetKey(id models.Snowflake) (*models.Key, error)
	DeleteKey(id models.Snowflake) error
	ListKeys(opts *ListKeyOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
	GetScopeEnum(scope string) (*models.ScopeEnum, error)
}

type ListKeyOptions struct {
	MarketplaceSlug string
}
