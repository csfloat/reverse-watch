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
	return k.conn.Model(&models.Key{}).Create(key).Error
}

func (k *keyRepository) Read(id models.Snowflake) (*models.Key, error) {
	var key models.Key
	err := k.conn.Model(&models.Key{}).
		Preload("Marketplace").
		Where("id = ?", id).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (k *keyRepository) Update(id models.Snowflake, opts *service.UpdateKeyOptions) error {
	return nil
}

func (k *keyRepository) Delete(id models.Snowflake) error {
	return k.conn.Model(&models.Key{}).Where("id = ?", id).Delete(&models.Key{}).Error
}

func (k *keyRepository) List(opts *service.ListKeyOptions) ([]*models.Key, error) {
	if opts == nil {
		return nil, errors.New(errors.InternalServerError, "ListKeyOptions is required")
	}

	keys := make([]*models.Key, 0)
	err := k.conn.Model(&models.Key{}).
		Where("marketplace_slug = ?", opts.MarketplaceSlug).
		Order("created_at DESC").
		Find(&keys).Error
	if err != nil {
		return nil, err
	}

	return keys, nil
}
