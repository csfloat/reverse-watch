package private

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"

	"gorm.io/gorm"
)

type keyRepository struct {
	conn *gorm.DB
}

var _ repository.KeyRepository = (*keyRepository)(nil)

func NewKeyRepository(conn *gorm.DB) repository.KeyRepository {
	return &keyRepository{
		conn: conn,
	}
}

func (k *keyRepository) Create(key *models.Key) error {
	return k.conn.Table("keys").Create(key).Error
}

func (k *keyRepository) Read(id models.Snowflake) (*models.Key, error) {
	var key models.Key
	err := k.conn.Table("keys").
		Preload("Marketplace").
		Preload("ScopeDetail").
		Where("id = ?", id).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (k *keyRepository) Delete(id models.Snowflake) error {
	return k.conn.Table("keys").Where("id = ?", id).Delete(&models.Key{}).Error
}

func (k *keyRepository) List(opts *service.ListKeyOptions) ([]*models.Key, error) {
	if opts == nil {
		return nil, errors.New(errors.InternalServerError, "ListKeyOptions is required")
	}

	keys := make([]*models.Key, 0)
	err := k.conn.Table("keys").
		Where("marketplace_slug = ?", opts.MarketplaceSlug).
		Order("created_at DESC").
		Find(&keys).Error
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (k *keyRepository) GetScopeEnum(scope string) (*models.ScopeEnum, error) {
	var scopeEnum models.ScopeEnum
	if err := k.conn.Table("scope_enums").Where("name = ?", scope).First(&scopeEnum).Error; err != nil {
		return nil, err
	}
	return &scopeEnum, nil
}
