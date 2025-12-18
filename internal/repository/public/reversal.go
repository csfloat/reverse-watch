package public

import (
	"time"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

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

func (r *reversalRepository) Expunge(id models.Snowflake) error {
	return r.conn.Model(&models.Reversal{}).Where("id = ?", id).Update("expunged_at", time.Now().UnixMilli()).Error
}

func (r *reversalRepository) buildListQuery(opts *repository.ReversalListOptions) *gorm.DB {
	query := r.conn.Model(&models.Reversal{}).Order("reversed_at DESC")
	if opts == nil {
		return query
	}
	if opts.SteamID.IsValid() {
		query = query.Where("steam_id = ?", opts.SteamID)
	}
	if opts.MarketplaceSlug != nil && *opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", opts.MarketplaceSlug)
	}
	if opts.Cursor != nil {
		query = query.Where("reversed_at <= ? AND id < ?", opts.Cursor.ReversedAt, opts.Cursor.ID)
	}
	if opts.Limit != nil {
		query = query.Limit(int(*opts.Limit))
	}
	return query
}

func (r *reversalRepository) List(opts *repository.ReversalListOptions) ([]*models.Reversal, error) {
	query := r.buildListQuery(opts)
	var reversals []*models.Reversal
	if err := query.Find(&reversals).Error; err != nil {
		return nil, err
	}
	return reversals, nil
}
