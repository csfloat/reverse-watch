package marketplace

import (
	"fmt"

	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/domain/service"
)

type marketplaceService struct {
	repository.PrivateRepository
}

var _ service.MarketplaceService = (*marketplaceService)(nil)

func NewMarketplaceService(repo repository.PrivateRepository) service.MarketplaceService {
	return &marketplaceService{
		PrivateRepository: repo,
	}
}

func (m *marketplaceService) CreateMarketplace(marketplace *models.Marketplace) error {
	return m.Marketplace().Create(marketplace)
}

func (m *marketplaceService) GetMarketplace(slug string) (*models.Marketplace, error) {
	return m.Marketplace().Read(slug)
}

func (m *marketplaceService) UpdateMarketplace(slug string, updates *dto.MarketplaceUpdates) error {
	if err := updates.Validate(); err != nil {
		return fmt.Errorf("invalid marketplace updates: %w", err)
	}
	return m.Marketplace().Update(slug, updates)
}

func (m *marketplaceService) DeleteMarketplace(slug string) error {
	return m.Marketplace().Delete(slug)
}
