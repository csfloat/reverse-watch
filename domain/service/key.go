package service

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
)

type KeyService interface {
	CreateKey(marketplaceSlug string, permissions models.Permissions) (*dto.RawKey, error)
	GetKey(id string) (*models.Key, error)
	DeleteKey(id string) error
	ListKeys(opts *dto.KeyListOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}
