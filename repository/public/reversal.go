package public

import (
	"time"

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
	if opts.ExcludeExpunged {
		query = query.Where("expunged_at IS NULL")
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

func (r *reversalRepository) SummaryStats() (*dto.SummaryStats, error) {
	cutoffMs := uint64(time.Now().UnixMilli() - 24*60*60*1000)

	var stats dto.SummaryStats
	err := r.conn.Raw(`
		SELECT
			COUNT(DISTINCT steam_id) AS traders_indexed,
			COUNT(DISTINCT steam_id) FILTER (WHERE expunged_at IS NULL) AS traders_flagged,
			COUNT(DISTINCT steam_id) FILTER (WHERE expunged_at IS NULL AND created_at >= ?) AS traders_flagged24h
		FROM reversals
		WHERE deleted_at IS NULL
	`, cutoffMs).Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *reversalRepository) DailyCounts(days int) ([]dto.DailyCount, error) {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	windowStart := today.AddDate(0, 0, -(days - 1))

	var rows []dto.DailyCount
	err := r.conn.Raw(`
		SELECT
			to_char(to_timestamp(reversed_at / 1000) AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS date,
			COUNT(*) AS count
		FROM reversals
		WHERE deleted_at IS NULL
		  AND expunged_at IS NULL
		  AND reversed_at >= ?
		GROUP BY date
		ORDER BY date ASC
	`, uint64(windowStart.UnixMilli())).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byDate := make(map[string]uint64, len(rows))
	for _, row := range rows {
		byDate[row.Date] = row.Count
	}

	result := make([]dto.DailyCount, 0, days)
	for d := windowStart; !d.After(today); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		result = append(result, dto.DailyCount{Date: key, Count: byDate[key]})
	}
	return result, nil
}
