package marketplace

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
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

func (m *marketplaceService) UpdateMarketplace(slug string, fields map[string]interface{}) error {
	return m.Marketplace().Update(slug, fields)
}

func (m *marketplaceService) DeleteMarketplace(slug string) error {
	return m.Marketplace().Delete(slug)
}
