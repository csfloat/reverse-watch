package service

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type KeyService interface {
	CreateKey(marketplaceSlug string, permissions models.Permissions) (*dto.RawKey, error)
	GetKey(id models.Snowflake) (*models.Key, error)
	UpdateKey(id models.Snowflake, opts *dto.KeyUpdateOptions) error
	DeleteKey(id models.Snowflake) error
	ListKeys(opts *dto.KeyListOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}
