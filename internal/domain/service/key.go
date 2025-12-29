package service

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
)

type KeyService interface {
	CreateKey(marketplaceSlug string, permissions models.Permissions) (*models.RawKey, error)
	GetKey(id string) (*models.Key, error)
	UpdateKey(id string, opts *repository.UpdateKeyOptions) error
	DeleteKey(id string) error
	ListKeys(opts *repository.KeyListOptions) ([]*models.Key, error)
	ValidateKey(secretKey string) (*models.Key, error)
}
