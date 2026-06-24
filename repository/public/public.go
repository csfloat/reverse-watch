package public

import (
	"reverse-watch/domain/models"

	"gorm.io/gorm"
)

func MigrateModels(tx *gorm.DB) error {
	publicModels := []interface{}{
		(*models.Reversal)(nil),
		(*models.SearchCount)(nil),
	}

	for _, model := range publicModels {
		if err := tx.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func CreateIndexes(tx *gorm.DB) error {
	indexes := []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_reversals_steam_id_marketplace_slug ON reversals(steam_id, marketplace_slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS idx_reversals_marketplace_slug ON reversals(marketplace_slug)`,
	}

	for _, index := range indexes {
		if err := tx.Exec(index).Error; err != nil {
			return err
		}
	}
	return nil
}
