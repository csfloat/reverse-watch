package repository

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
)

type KeyRepository interface {
	Create(key *models.Key) error
	Read(id models.Snowflake) (*models.Key, error)
	Delete(id models.Snowflake) error
	List(opts *service.ListKeyOptions) ([]*models.Key, error)
	GetScopeEnum(scope string) (*models.ScopeEnum, error)
}

type MarketplaceRepository interface {
	Create(marketplace *models.Marketplace) error
	Read(slug string) (*models.Marketplace, error)
	Update(slug string, fields map[string]interface{}) error
	Delete(slug string) error
}

type PrivateRepository interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
}
