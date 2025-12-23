package service

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
)

type KeyService interface {
	CreateKey(marketplaceSlug string, permissions models.Permissions) (*models.RawKey, error)
	GetKey(id models.Snowflake) (*models.Key, error)
	DeleteKey(id models.Snowflake) error
	ListKeys(opts *repository.KeyListOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}
