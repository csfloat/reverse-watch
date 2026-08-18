package public

import (
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"

	"gorm.io/gorm"
)

type searchCountRepository struct {
	conn *gorm.DB
}

var _ repository.SearchCountRepository = (*searchCountRepository)(nil)

func NewSearchCountRepository(conn *gorm.DB) repository.SearchCountRepository {
	return &searchCountRepository{
		conn: conn,
	}
}

func (r *searchCountRepository) Increment(steamID models.SteamID) error {
	now := uint64(time.Now().UnixMilli())
	return r.conn.Exec(`
		INSERT INTO search_counts (steam_id, count, last_searched_at)
		VALUES (?, 1, ?)
		ON CONFLICT (steam_id) DO UPDATE
		SET count = search_counts.count + 1,
		    last_searched_at = EXCLUDED.last_searched_at
	`, uint64(steamID), now).Error
}
