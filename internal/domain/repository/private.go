package repository

import (
	"reverse-watch/internal/domain/models"
)

type KeyListOptions struct {
	MarketplaceSlug *string
}

type KeyRepository interface {
	Create(key *models.Key) error
	Read(id models.Snowflake) (*models.Key, error)
	Delete(id models.Snowflake) error
	List(opts *KeyListOptions) ([]*models.Key, error)
}

type MarketplaceRepository interface {
	Create(marketplace *models.Marketplace) error
	Read(slug string) (*models.Marketplace, error)
	Update(slug string, fields map[string]interface{}) error
	Delete(slug string) error
}

type AdminAuditListOptions struct {
	TargetActions  []models.TargetAction
	TargetResource *models.Snowflake
}

type AdminAuditRepository interface {
	Create(audit *models.AdminAudit) error
	Read(id models.Snowflake) (*models.AdminAudit, error)
	Delete(id models.Snowflake) error
	List(opts *AdminAuditListOptions) ([]*models.AdminAudit, error)
}

type PrivateRepository interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
}
