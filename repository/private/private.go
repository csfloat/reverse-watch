package private

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/domain/repository"
	"reverse-watch/domain/secret"
	"reverse-watch/logging"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var (
	once   sync.Once
	mu     sync.RWMutex
	closed bool
	err    error

	privateRepo repository.PrivateRepository = (*privateRepository)(nil)
)

type privateRepository struct {
	conn *gorm.DB

	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	key         repository.KeyRepository
}

func NewPrivateRepository(cfg config.Config, keygen secret.KeyGenerator) (repository.PrivateRepository, error) {
	once.Do(func() {
		rootDir, innerErr := config.GetProjectRootDir()
		if innerErr != nil {
			err = innerErr
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "private.db")
		conn, innerErr := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if innerErr != nil {
			err = innerErr
			return
		}

		repo := &privateRepository{
			conn:        conn,
			key:         NewKeyRepository(conn),
			marketplace: NewMarketplaceRepository(conn),
			adminAudit:  NewAdminAuditRepository(conn),
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

		if innerErr := migratePrivateModels(conn); innerErr != nil {
			err = onErr(innerErr)
			return
		}

		if innerErr := seedMarketplaces(conn); innerErr != nil {
			err = onErr(innerErr)
			return
		}

		if innerErr := seedAdminAPIKey(conn, keygen); innerErr != nil {
			err = onErr(innerErr)
			return
		}

		privateRepo = repo
	})

	if err != nil {
		return nil, err
	}

	mu.RLock()
	defer mu.RUnlock()

	if closed {
		return nil, fmt.Errorf("repository already closed")
	}
	return privateRepo, nil
}

func (p *privateRepository) Close() error {
	db, err := p.conn.DB()
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	closed = true
	return db.Close()
}

func (p *privateRepository) Key() repository.KeyRepository {
	return p.key
}

func (p *privateRepository) Marketplace() repository.MarketplaceRepository {
	return p.marketplace
}

func (p *privateRepository) AdminAudit() repository.AdminAuditRepository {
	return p.adminAudit
}

func migratePrivateModels(tx *gorm.DB) error {
	privateModels := []interface{}{
		(*models.Marketplace)(nil),
		(*models.Key)(nil),
		(*models.AdminAudit)(nil),
	}

	for _, model := range privateModels {
		if err := tx.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func seedMarketplaces(tx *gorm.DB) error {
	marketplace := &models.Marketplace{
		Slug:     "csfloat",
		Name:     "CSFloat",
		IsActive: true,
	}

	return tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(marketplace).Error
}

func seedAdminAPIKey(tx *gorm.DB, keygen secret.KeyGenerator) error {
	var exists bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM keys WHERE permissions & ? = ? AND environment = ?)`, models.PermissionAdmin, models.PermissionAdmin, keygen.Environment()).
		Row().Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	secretKey, err := keygen.GenerateSecretKey()
	if err != nil {
		return err
	}

	formattedKey, err := secretKey.Format()
	if err != nil {
		return err
	}

	id, err := secretKey.ID()
	if err != nil {
		return err
	}

	permissions := models.PermissionAdmin
	permissions.AddAllPermissions()

	adminKey := &models.Key{
		ID:              id,
		Environment:     keygen.Environment(),
		MarketplaceSlug: "csfloat",
		Permissions:     permissions,
	}

	if err := tx.Create(adminKey).Error; err != nil {
		return err
	}

	logging.Log.Infof("GENERATED ADMIN SECRET KEY: %s\n\nSAVE THIS FOR FUTURE PURPOSES, WON'T BE SHOWN AGAIN!", formattedKey)
	return nil
}
