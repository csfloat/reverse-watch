package private

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"

	"gorm.io/gorm"
)

type marketplaceRepository struct {
	conn *gorm.DB
}

var _ repository.MarketplaceRepository = (*marketplaceRepository)(nil)

func NewMarketplaceRepository(conn *gorm.DB) repository.MarketplaceRepository {
	return &marketplaceRepository{
		conn: conn,
	}
}

func (m *marketplaceRepository) Create(marketplace *models.Marketplace) error {
	return m.conn.Create(marketplace).Error
}

func (m *marketplaceRepository) Read(slug string) (*models.Marketplace, error) {
	var marketplace models.Marketplace
	if err := m.conn.Model(&models.Marketplace{}).Where("slug = ?", slug).First(&marketplace).Error; err != nil {
		return nil, err
	}
	return &marketplace, nil
}

func (m *marketplaceRepository) Update(slug string, updates *dto.MarketplaceUpdates) error {
	if err := updates.Validate(); err != nil {
		return err
	}

	tx := m.conn.Model(&models.Marketplace{}).Where("slug = ?", slug).Updates(updates)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *marketplaceRepository) Delete(slug string) error {
	return m.conn.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Key{}).Where("marketplace_slug = ?", slug).Delete(&models.Key{}).Error; err != nil {
			return err
		}

		result := tx.Model(&models.Marketplace{}).Where("slug = ?", slug).Delete(&models.Marketplace{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
