package public

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func (r *reversalRepository) create(reversals []*models.Reversal) error {
	return r.conn.Model(&models.Reversal{}).Create(reversals).Error
}

func (r *reversalRepository) Create(reversal *models.Reversal) error {
	return r.create([]*models.Reversal{reversal})
}

func (r *reversalRepository) BulkCreate(reversals []*models.Reversal) error {
	return r.create(reversals)
}

func (r *reversalRepository) Read(id models.Snowflake) (*models.Reversal, error) {
	var reversal models.Reversal
	if err := r.conn.Model(&models.Reversal{}).Where("id = ?", id).First(&reversal).Error; err != nil {
		return nil, err
	}
	return &reversal, nil
}

func (r *reversalRepository) Update(id models.Snowflake, updates *dto.ReversalUpdates) error {
	if err := updates.Validate(); err != nil {
		return err
	}

	tx := r.conn.Model(&models.Reversal{}).Where("id = ?", id).Updates(updates.ToFields())
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
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

func (r *reversalRepository) DeleteAllUserReports(steamId models.SteamID) error {
	tx := r.conn.Unscoped().Where("steam_id = ?", steamId).Delete(&models.Reversal{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *reversalRepository) buildListQuery(opts *dto.ReversalListOptions) *gorm.DB {
	query := r.conn.Model(&models.Reversal{})
	if opts == nil {
		return query
	}

	var desc bool
	if opts.OrderParam != nil {
		desc = opts.OrderParam.Direction == dto.DESC
		orderBy := clause.OrderByColumn{
			Column: clause.Column{Name: opts.OrderParam.Column},
			Desc:   desc,
		}

		query = query.Order(orderBy)
	}

	if opts.SteamID.IsValid() {
		query = query.Where("steam_id = ?", opts.SteamID)
	}
	if opts.MarketplaceSlug != nil && *opts.MarketplaceSlug != "" {
		query = query.Where("marketplace_slug = ?", opts.MarketplaceSlug)
	}
	if opts.Cursor != nil {
		// Adjust direction based on order specified
		if desc {
			query = query.Where("id < ?", opts.Cursor.ID)
		} else {
			query = query.Where("id > ?", opts.Cursor.ID)
		}
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
