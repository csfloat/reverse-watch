package service

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
)

type MarketplaceService interface {
	CreateMarketplace(marketplace *models.Marketplace) error
	GetMarketplace(slug string) (*models.Marketplace, error)
	UpdateMarketplace(slug string, updates *dto.MarketplaceUpdates) error
	DeleteMarketplace(slug string) error
}
