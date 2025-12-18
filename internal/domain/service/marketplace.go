package service

import "reverse-watch/internal/domain/models"

type MarketplaceService interface {
	CreateMarketplace(marketplace *models.Marketplace) error
	GetMarketplace(slug string) (*models.Marketplace, error)
	UpdateMarketplace(slug string, fields map[string]interface{}) error
	DeleteMarketplace(slug string) error
}
