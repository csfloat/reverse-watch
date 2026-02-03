package public

import (
	"errors"
	"path/filepath"
	"sync"

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/logging"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	once    sync.Once
	initErr error
)

type publicRepository struct {
	db *gorm.DB
}

func NewPublicRepository(cfg config.Config) (*publicRepository, error) {
	var public *publicRepository
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

		public = &publicRepository{
			db: conn,
		}

		onErr := func(err error) error {
			if closeErr := public.Close(); closeErr != nil {
				logging.Log.Errorf("failed to close public repository: %v", closeErr)
				return errors.Join(err, closeErr)
			}
			return err
		}

		if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
			initErr = onErr(err)
			return
		}

		if err := migratePublicModels(conn); err != nil {
			initErr = onErr(err)
			return
		}

		if err := createIndexes(conn); err != nil {
			initErr = onErr(err)
			return
		}
	})
	if initErr != nil {
		return nil, initErr
	}
	return public, nil
}

func (p *publicRepository) Close() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
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
