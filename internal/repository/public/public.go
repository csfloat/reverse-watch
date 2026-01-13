package public

import (
	"path/filepath"

	"reverse-watch/internal/config"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type publicRepository struct {
	conn *gorm.DB
}

var _ repository.PublicRepository = (*publicRepository)(nil)

func NewPublicRepository(cfg config.Config) (repository.PublicRepository, error) {
	rootDir, err := config.GetProjectRootDir()
	if err != nil {
		return nil, err
	}

	dsn := filepath.Join(rootDir, cfg.StaticDir, "public.db")
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
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

	return &publicRepository{
		conn: conn,
	}, nil
}

func (p *publicRepository) Reversal() repository.ReversalRepository {
	// STUB
	return nil
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
		`CREATE INDEX IF NOT EXISTS idx_reversals_reversed_at_desc ON reversals(reversed_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_reversals_steam_id_reversed_at_desc ON reversals(steam_id, reversed_at DESC, id DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_reversals_marketplace_slug_reversed_at_desc ON reversals(marketplace_slug, reversed_at DESC, id DESC)`,
	}

	for _, index := range indexes {
		if err := tx.Exec(index).Error; err != nil {
			return err
		}
	}
	return nil
}
