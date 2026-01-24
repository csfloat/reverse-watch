package repository

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type KeyRepository interface {
	Create(key *models.Key) error
	Read(id string) (*models.Key, error)
	Delete(id string) error
	List(opts *dto.KeyListOptions) ([]*models.Key, error)
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

type PrivateRepository interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
	Close() error
}
