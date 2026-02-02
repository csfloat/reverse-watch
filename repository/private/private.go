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
	once sync.Once
	err  error
)

func NewPrivateRepository(cfg config.Config, keygen secret.KeyGenerator) (*gorm.DB, error) {
	var db *gorm.DB
	once.Do(func() {
		rootDir, error := config.GetProjectRootDir()
		if error != nil {
			err = error
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "private.db")
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

		if error := migratePrivateModels(conn); error != nil {
			err = error
			return
		}

		if error := seedMarketplaces(conn); error != nil {
			err = error
			return
		}

		if error := seedAdminAPIKey(conn, keygen); error != nil {
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
