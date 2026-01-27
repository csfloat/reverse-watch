package private

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/domain/secret"

	"gorm.io/gorm"
)

type keyRepository struct {
	conn   *gorm.DB
	keygen secret.KeyGenerator
}

var _ repository.KeyRepository = (*keyRepository)(nil)

func NewKeyRepository(conn *gorm.DB, keygen secret.KeyGenerator) repository.KeyRepository {
	return &keyRepository{
		conn:   conn,
		keygen: keygen,
	}
}

func (k *keyRepository) Create(marketplaceSlug string, permissions models.Permissions) (*dto.RawKey, error) {
	secretKey, err := k.keygen.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	id, err := secretKey.ID()
	if err != nil {
		return nil, err
	}

	key := &models.Key{
		ID:              id,
		Environment:     k.keygen.Environment(),
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}
	if err := k.conn.Model(&models.Key{}).Create(key).Error; err != nil {
		return nil, err
	}

	formattedKey, err := secretKey.Format()
	if err != nil {
		return nil, err
	}

	return &dto.RawKey{
		ID:              id,
		Environment:     k.keygen.Environment(),
		SecretKey:       formattedKey,
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}, nil
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

func (k *keyRepository) ValidateKey(secretKey string) (*models.Key, error) {
	hashedKey := secret.Sha256Hash(secretKey)
	return k.Read(hashedKey)
}
