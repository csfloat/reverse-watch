package private

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"

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

func (k *keyRepository) Read(id string) (*models.Key, error) {
	var key models.Key
	err := k.conn.Model(&models.Key{}).
		Preload("Marketplace").
		Where("id = ?", id).First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

func (k *keyRepository) Delete(id string) error {
	tx := k.conn.Model(&models.Key{}).Where("id = ?", id).Delete(&models.Key{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (k *keyRepository) buildListQuery(opts *dto.KeyListOptions) *gorm.DB {
	query := k.conn.Model(&models.Key{})
	if opts == nil {
		return query
	}
	if opts.MarketplaceSlug != nil && *opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", *opts.MarketplaceSlug)
	}
	return query
}

func (k *keyRepository) List(opts *dto.KeyListOptions) ([]*models.Key, error) {
	query := k.buildListQuery(opts)

	var keys []*models.Key
	if err := query.Preload("Marketplace").Order("id DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}
