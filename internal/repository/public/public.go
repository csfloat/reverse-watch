package public

import (
	"errors"
	"path/filepath"
	"sync"

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

var (
	once       sync.Once
	publicRepo repository.PublicRepository = (*publicRepository)(nil)
)

func NewPublicRepository(cfg config.Config) (repository.PublicRepository, error) {
	var err error

	once.Do(func() {
		rootDir, innerErr := config.GetProjectRootDir()
		if innerErr != nil {
			err = innerErr
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "public.db")
		conn, innerErr := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if innerErr != nil {
			err = innerErr
			return
		}

		repo := &publicRepository{
			conn: conn,
		}

		onErr := func(err error) error {
			if innerErr := repo.Close(); innerErr != nil {
				return errors.Join(err, innerErr)
			}
			return err
		}

		if innerErr := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; innerErr != nil {
			err = onErr(innerErr)
			return
		}

		if innerErr := migratePublicModels(conn); innerErr != nil {
			err = onErr(innerErr)
			return
		}

		if innerErr := createIndexes(conn); innerErr != nil {
			err = onErr(innerErr)
			return
		}

		publicRepo = repo
	})
	if err != nil {
		return nil, err
	}
	return publicRepo, nil
}

func (p *publicRepository) Close() error {
	db, err := p.conn.DB()
	if err != nil {
		return err
	}
	return db.Close()
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
