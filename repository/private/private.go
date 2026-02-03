package private

import (
	"errors"
	"path/filepath"
	"sync"

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/domain/secret"
	"reverse-watch/logging"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var (
	once    sync.Once
	initErr error
)

type privateRepository struct {
	db *gorm.DB
}

func NewPrivateRepository(cfg config.Config, keygen secret.KeyGenerator) (*privateRepository, error) {
	var private *privateRepository
	once.Do(func() {
		rootDir, err := config.GetProjectRootDir()
		if err != nil {
			initErr = err
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "private.db")
		conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			initErr = err
			return
		}

		private = &privateRepository{
			db: conn,
		}

		onErr := func(err error) error {
			if closeErr := private.Close(); closeErr != nil {
				logging.Log.Errorf("failed to close private repository: %v", closeErr)
				return errors.Join(err, closeErr)
			}
			return err
		}

		if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
			initErr = onErr(err)
			return
		}

		if err := migratePrivateModels(conn); err != nil {
			initErr = onErr(err)
			return
		}

		if err := seedMarketplaces(conn); err != nil {
			initErr = onErr(err)
			return
		}

		if err := seedAdminAPIKey(conn, keygen); err != nil {
			initErr = onErr(err)
			return
		}
	})

	if initErr != nil {
		return nil, initErr
	}
	return private, nil
}

func (p *privateRepository) Close() error {
	db, err := p.db.DB()
	if err != nil {
		return err
	}
	return db.Close()
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
