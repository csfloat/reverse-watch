package repository

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
)

type KeyRepository interface {
	Create(marketplaceSlug string, permissions models.Permissions) (*dto.RawKey, error)
	Read(id string) (*models.Key, error)
	Delete(id string) error
	List(opts *dto.KeyListOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}

type MarketplaceRepository interface {
	Create(marketplace *models.Marketplace) error
	Read(slug string) (*models.Marketplace, error)
	Update(slug string, updates *dto.MarketplaceUpdates) error
	Delete(slug string) error
}

type AdminAuditRepository interface {
	Create(audit *models.AdminAudit) error
	Read(id models.Snowflake) (*models.AdminAudit, error)
	Delete(id models.Snowflake) error
	List(opts *dto.AdminAuditListOptions) ([]*models.AdminAudit, error)
}
