package public

import (
	"errors"
	"fmt"
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
	once   sync.Once
	mu     sync.RWMutex
	closed bool
	err    error

	publicRepo repository.PublicRepository = (*publicRepository)(nil)
)

func NewPublicRepository(cfg config.Config) (repository.PublicRepository, error) {
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

			mu.Lock()
			defer mu.Unlock()

			closed = true
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

	mu.RLock()
	defer mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("repository already closed")
	}
	return publicRepo, nil
}

func (p *publicRepository) Close() error {
	db, err := p.conn.DB()
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	closed = true
	return db.Close()
}

func (p *publicRepository) Reversal() repository.ReversalRepository {
	return NewReversalRepository(p.conn)
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
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_reversals_steam_id_marketplace_slug ON reversals(steam_id, marketplace_slug)`,
		`CREATE INDEX IF NOT EXISTS idx_reversals_marketplace_slug ON reversals(marketplace_slug)`,
		`CREATE INDEX IF NOT EXISTS idx_reversals_created_at_desc ON reversals(created_at DESC, id DESC)`,
	}

	for _, index := range indexes {
		if err := tx.Exec(index).Error; err != nil {
			return err
		}
	}
	return nil
}
