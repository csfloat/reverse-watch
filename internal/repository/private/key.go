package private

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

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

func (k *keyRepository) Update(id models.Snowflake, opts *repository.KeyUpdateOptions) error {
	return nil
}

func (k *keyRepository) Delete(id models.Snowflake) error {
	return k.conn.Model(&models.Key{}).Where("id = ?", id).Delete(&models.Key{}).Error
}

func (k *keyRepository) buildListQuery(opts *repository.KeyListOptions) *gorm.DB {
	query := k.conn.Model(&models.Key{})
	if opts == nil {
		return query
	}
	if opts.MarketplaceSlug != nil && *opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", *opts.MarketplaceSlug)
	}
	return query
}

func (k *keyRepository) List(opts *repository.KeyListOptions) ([]*models.Key, error) {
	query := k.buildListQuery(opts)

	var keys []*models.Key
	if err := query.Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}
