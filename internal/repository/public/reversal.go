package public

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"

	"gorm.io/gorm"
)

type reversalRepository struct {
	conn *gorm.DB
}

var _ repository.ReversalRepository = (*reversalRepository)(nil)

func NewReversalRepository(conn *gorm.DB) repository.ReversalRepository {
	return &reversalRepository{
		conn: conn,
	}
}

func (r *reversalRepository) Create(reversal *models.Reversal) error {
	return r.conn.Model(&models.Reversal{}).Create(reversal).Error
}

func (r *reversalRepository) BulkCreate(reversals []*models.Reversal) error {
	return r.conn.Model(&models.Reversal{}).Create(reversals).Error
}

func (r *reversalRepository) Read(id models.Snowflake) (*models.Reversal, error) {
	var reversal models.Reversal
	if err := r.conn.Model(&models.Reversal{}).Where("id = ?", id).First(&reversal).Error; err != nil {
		return nil, err
	}
	return &reversal, nil
}

func (r *reversalRepository) Update(id models.Snowflake, fields map[string]interface{}) error {
	return r.conn.Model(&models.Reversal{}).Where("id = ?", id).Updates(fields).Error
}

func (r *reversalRepository) Delete(id models.Snowflake) error {
	return r.conn.Model(&models.Reversal{}).Where("id = ?", id).Delete(&models.Reversal{}).Error
}

func (r *reversalRepository) buildListQuery(opts *service.ListReversalOptions) *gorm.DB {
	query := r.conn.Model(&models.Reversal{})
	if opts.SteamID.IsValid() {
		query = query.Where("steam_id = ?", opts.SteamID)
	}
	if opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", opts.MarketplaceSlug)
	}
	return query
}

func (r *reversalRepository) List(opts *service.ListReversalOptions) ([]*models.Reversal, error) {
	if opts == nil {
		return nil, errors.New(errors.InternalServerError, "ListReversalOptions is required")
	}

	query := r.buildListQuery(opts)

	reversals := make([]*models.Reversal, 0)
	err := query.Order("reversed_at DESC").Find(&reversals).Error
	if err != nil {
		return nil, err
	}
	return reversals, nil
}
