package private

import (
	"fmt"

	"reverse-watch/internal/domain/dto"
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
	return m.conn.Create(marketplace).Error
}

func (m *marketplaceRepository) Read(slug string) (*models.Marketplace, error) {
	var marketplace models.Marketplace
	if err := m.conn.Model(&models.Marketplace{}).Where("slug = ?", slug).First(&marketplace).Error; err != nil {
		return nil, err
	}
	return &marketplace, nil
}

func (m *marketplaceRepository) Update(slug string, opts *dto.MarketplaceUpdateOptions) error {
	if opts == nil {
		return fmt.Errorf("marketplace update options cannot be nil")
	}

	fields := opts.ToFields()
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}

	tx := m.conn.Model(&models.Marketplace{}).Where("slug = ?", slug).Updates(fields)
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (m *marketplaceRepository) Delete(slug string) error {
	tx := m.conn.Model(&models.Marketplace{}).Where("slug = ?", slug).Delete(&models.Marketplace{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
