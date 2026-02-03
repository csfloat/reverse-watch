package public

import (
	"path/filepath"
	"sync"

	"reverse-watch/config"
	"reverse-watch/domain/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once    sync.Once
	initErr error
)

func NewPublicRepository(cfg config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	once.Do(func() {
		rootDir, err := config.GetProjectRootDir()
		if err != nil {
			initErr = err
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "public.db")
		conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			initErr = err
			return
		}

		if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
			initErr = err
			return
		}

		if err := migratePublicModels(conn); err != nil {
			initErr = err
			return
		}

		if err := createIndexes(conn); err != nil {
			initErr = err
			return
		}

		db = conn
	})
	if initErr != nil {
		return nil, initErr
	}
	return db, nil
}

func migratePublicModels(tx *gorm.DB) error {
	publicModels := []interface{}{
		(*models.Reversal)(nil),
	}

	for _, model := range publicModels {
		if err := tx.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func createIndexes(tx *gorm.DB) error {
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
