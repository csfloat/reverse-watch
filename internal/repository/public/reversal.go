package public

import (
	"fmt"

	"reverse-watch/internal/domain/dto"
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

func (r *reversalRepository) Create(reversals ...*models.Reversal) error {
	return r.conn.Model(&models.Reversal{}).Create(reversals).Error
}

func (r *reversalRepository) Read(id models.Snowflake) (*models.Reversal, error) {
	var reversal models.Reversal
	if err := r.conn.Model(&models.Reversal{}).Where("id = ?", id).First(&reversal).Error; err != nil {
		return nil, err
	}
	return &reversal, nil
}

func (r *reversalRepository) Update(id models.Snowflake, opts *dto.ReversalUpdateOptions) error {
	if opts == nil {
		return fmt.Errorf("opts cannot be nil")
	}

	if err := opts.Validate(); err != nil {
		return err
	}

	fields := opts.ToFields()
	if len(fields) == 0 {
		return fmt.Errorf("no fields to update")
	}
	return r.conn.Model(&models.Reversal{}).Where("id = ?", id).Updates(fields).Error
}

func (r *reversalRepository) Delete(id models.Snowflake) error {
	tx := r.conn.Where("id = ?", id).Delete(&models.Reversal{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *reversalRepository) buildListQuery(opts *dto.ReversalListOptions) *gorm.DB {
	query := r.conn.Model(&models.Reversal{}).Order("created_at DESC, id DESC")
	if opts == nil {
		return query
	}
	if opts.SteamID.IsValid() {
		query = query.Where("steam_id = ? OR related_steam_id = ?", opts.SteamID, opts.SteamID)
	}
	if opts.MarketplaceSlug != nil && *opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", opts.MarketplaceSlug)
	}
	if opts.Cursor != nil {
		query = query.Where("(created_at, id) < (?, ?)", opts.Cursor.CreatedAt, opts.Cursor.ID)
	}
	if opts.Limit != nil {
		query = query.Limit(int(*opts.Limit))
	}
	return query
}

func (r *reversalRepository) List(opts *dto.ReversalListOptions) ([]*models.Reversal, error) {
	query := r.buildListQuery(opts)
	var reversals []*models.Reversal
	if err := query.Find(&reversals).Error; err != nil {
		return nil, err
	}
	return reversals, nil
}
