package service

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type MarketplaceService interface {
	CreateMarketplace(marketplace *models.Marketplace) error
	GetMarketplace(slug string) (*models.Marketplace, error)
	UpdateMarketplace(slug string, updates *dto.MarketplaceUpdates) error
	DeleteMarketplace(slug string) error
}
