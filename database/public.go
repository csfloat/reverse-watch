package database

import (
	"reverse-watch/config"
	"reverse-watch/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitializePublicDB(cfg config.Config) (*gorm.DB, error) {
	conn, err := gorm.Open(sqlite.Open(cfg.PublicDB.Filename), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, err
	}

	if err := migratePublicModels(conn); err != nil {
		return nil, err
	}

	if err := createIndexes(conn); err != nil {
		return nil, err
	}

	return conn, nil
}

func migratePublicModels(tx *gorm.DB) error {
	models := []interface{}{
		(*types.Reversal)(nil),
	}

	for _, model := range models {
		if err := tx.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func createIndexes(tx *gorm.DB) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_reversals_steam_id_reversed_at_desc ON reversals(steam_id, reversed_at DESC)`,
	}

	for _, index := range indexes {
		if err := tx.Exec(index).Error; err != nil {
			return err
		}
	}
	return nil
}
