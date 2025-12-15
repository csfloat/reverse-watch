package repository

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
)

type KeyRepository interface {
	Create(key *models.Key) error
	Read(id models.Snowflake) (*models.Key, error)
	Update(id models.Snowflake, opts service.UpdateKeyOptions) error
	Delete(id models.Snowflake) error
	List(opts service.KeyListOptions) ([]*models.Key, error)
}

type MarketplaceRepository interface {
	Create(marketplace *models.Marketplace) error
	Read(slug string) (*models.Marketplace, error)
	Update(slug string, fields map[string]interface{}) error
	Delete(slug string) error
}

type AdminAuditRepository interface {
	Create(audit *models.AdminAudit) error
	Read(id models.Snowflake) (*models.AdminAudit, error)
	Delete(id models.Snowflake) error
	List(opts service.AdminAuditListOptions) ([]*models.AdminAudit, error)
}

type PrivateRepository interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
}
