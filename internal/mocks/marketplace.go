package mocks

import (
	"fmt"
	"sync"
	"time"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

	"gorm.io/gorm"
)

type MockMarketplaceRepository struct {
	mutex        sync.RWMutex
	marketplaces map[string]*models.Marketplace
}

var _ repository.MarketplaceRepository = (*MockMarketplaceRepository)(nil)

func NewMockMarketplaceRepository() repository.MarketplaceRepository {
	return &MockMarketplaceRepository{
		marketplaces: make(map[string]*models.Marketplace),
	}
}

func cloneMarketplace(marketplace *models.Marketplace) *models.Marketplace {
	return &models.Marketplace{
		Slug:      marketplace.Slug,
		CreatedAt: marketplace.CreatedAt,
		UpdatedAt: marketplace.UpdatedAt,
		Name:      marketplace.Name,
		IsActive:  marketplace.IsActive,
	}
}

func (m *MockMarketplaceRepository) Create(marketplace *models.Marketplace) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.marketplaces[marketplace.Slug]; ok {
		return fmt.Errorf("UNIQUE constraint failed: marketplaces.slug")
	}

	if err := marketplace.BeforeCreate(nil); err != nil {
		return err
	}

	now := uint64(time.Now().UnixMilli())
	marketplace.CreatedAt = now
	marketplace.UpdatedAt = now

	m.marketplaces[marketplace.Slug] = cloneMarketplace(marketplace)
	return nil
}

func (m *MockMarketplaceRepository) Read(slug string) (*models.Marketplace, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	marketplace, ok := m.marketplaces[slug]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return marketplace, nil
}

func (m *MockMarketplaceRepository) Update(slug string, updates *dto.MarketplaceUpdates) error {
	if updates == nil {
		return fmt.Errorf("updates cannot be nil")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	marketplace, ok := m.marketplaces[slug]
	if !ok {
		return gorm.ErrRecordNotFound
	}

	marketplace.UpdatedAt = uint64(time.Now().UnixMilli())

	if updates.Name != nil {
		marketplace.Name = *updates.Name
	}
	if updates.IsActive != nil {
		marketplace.IsActive = *updates.IsActive
	}
	m.marketplaces[slug] = marketplace

	return nil
}

func (m *MockMarketplaceRepository) Delete(slug string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.marketplaces[slug]; !ok {
		return gorm.ErrRecordNotFound
	}

	delete(m.marketplaces, slug)
	return nil
}
