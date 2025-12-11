package private

import (
	"reverse-watch/types"
)

func (s *Service) CreateMarketplace(marketplace *types.Marketplace) error {
	return s.conn.Table("marketplaces").Create(marketplace).Error
}

func (s *Service) UpdateMarketplace(slug string, fields map[string]interface{}) error {
	return s.conn.Table("marketplaces").Where("slug = ?", slug).Updates(fields).Error
}

func (s *Service) DeleteMarketplace(slug string) error {
	tx := s.conn.Begin()

	// Delete associated keys
	if err := tx.Table("keys").Where("marketplace_slug = ?", slug).Delete(&types.Key{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Table("marketplaces").Where("slug = ?", slug).Delete(&types.Marketplace{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (s *Service) GetMarketplace(slug string) (*types.Marketplace, error) {
	var marketplace types.Marketplace
	if err := s.conn.Table("marketplaces").Where("slug = ?", slug).First(&marketplace).Error; err != nil {
		return nil, err
	}
	return &marketplace, nil
}
