package public

import (
	"errors"
	"path/filepath"
	"sync"

	"reverse-watch/config"
	"reverse-watch/domain/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once sync.Once
	err  error
)

func NewPublicRepository(cfg config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	once.Do(func() {
		rootDir, error := config.GetProjectRootDir()
		if error != nil {
			err = error
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "public.db")
		conn, error := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if error != nil {
			err = error
			return
		}

		if error := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; error != nil {
			err = error
			return
		}

		if error := migratePublicModels(conn); error != nil {
			err = error
			return
		}

		if error := createIndexes(conn); error != nil {
			err = error
			return
		}

		db = conn
	})
	if err != nil {
		conn, error := db.DB()
		if error != nil {
			return nil, errors.Join(err, error)
		}
		conn.Close()
		return nil, err
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
