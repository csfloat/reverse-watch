package private

import (
	"errors"

	"reverse-watch/types"
)

func (s *Service) CreateKey(key *types.Key) error {
	return s.conn.Table("keys").Create(key).Error
}

func (s *Service) DeleteKey(id types.Snowflake) error {
	return s.conn.Table("keys").Where("id = ?", id).Delete(&types.Key{}).Error
}

func (s *Service) GetKeyFromID(id types.Snowflake) (*types.Key, error) {
	var key types.Key
	err := s.conn.Table("keys").
		Preload("Marketplace").
		Preload("ScopeDetail").
		Where("id = ?", id).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

type ListKeyOptions struct {
	MarketplaceSlug string
}

func (s *Service) ListKeys(opts *ListKeyOptions) ([]*types.Key, error) {
	if opts == nil {
		return nil, errors.New("ListKeyOptions is required")
	}

	keys := make([]*types.Key, 0)
	err := s.conn.Table("keys").
		Where("marketplace_slug = ?", opts.MarketplaceSlug).
		Order("created_at desc").
		Find(&keys).Error
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (s *Service) GetScopeEnum(scope string) (*types.ScopeEnum, error) {
	var scopeEnum types.ScopeEnum
	if err := s.conn.Table("scope_enums").Where("name = ?", scope).Find(&scopeEnum).Error; err != nil {
		return nil, err
	}
	return &scopeEnum, nil
}

func (s *Service) CreateMarketplace(marketplace *types.Marketplace) error {
	return s.conn.Table("marketplaces").Create(marketplace).Error
}
