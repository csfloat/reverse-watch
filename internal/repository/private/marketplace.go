package private

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

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
	return m.conn.Table("marketplaces").Create(marketplace).Error
}

func (m *marketplaceRepository) Read(slug string) (*models.Marketplace, error) {
	var marketplace models.Marketplace
	if err := m.conn.Table("marketplaces").Where("slug = ?", slug).First(&marketplace).Error; err != nil {
		return nil, err
	}
	return &marketplace, nil
}

func (m *marketplaceRepository) Update(slug string, fields map[string]interface{}) error {
	return m.conn.Table("marketplaces").Where("slug = ?", slug).Updates(fields).Error
}

func (m *marketplaceRepository) Delete(slug string) error {
	tx := m.conn.Begin()
	if err := tx.Table("keys").Where("marketplace_slug = ?", slug).Delete(&models.Key{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Table("marketplaces").Where("slug = ?", slug).Delete(&models.Marketplace{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}
