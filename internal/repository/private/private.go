package private

import (
	"path/filepath"
	"sync"

	"reverse-watch/internal/config"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/secret"
	"reverse-watch/internal/logging"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

var (
	once        sync.Once
	privateRepo repository.PrivateRepository = (*privateRepository)(nil)
)

type privateRepository struct {
	conn *gorm.DB
}

func NewPrivateRepository(cfg config.Config, keygen secret.KeyGenerator) (repository.PrivateRepository, error) {
	var error error

	once.Do(func() {
		rootDir, err := config.GetProjectRootDir()
		if err != nil {
			error = err
			return
		}

		dsn := filepath.Join(rootDir, cfg.StaticDir, "private.db")
		conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err != nil {
			error = err
			return
		}

		if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
			error = err
			return
		}

		if err := migratePrivateModels(conn); err != nil {
			error = err
			return
		}

		if err := seedMarketplaces(conn); err != nil {
			error = err
			return
		}

		if err := seedAdminAPIKey(conn, keygen); err != nil {
			error = err
			return
		}

		privateRepo = &privateRepository{
			conn: conn,
		}
	})

	if error != nil {
		return nil, error
	}
	return privateRepo, nil
}

func (p *privateRepository) Key() repository.KeyRepository {
	// STUB
	return nil
}

func (p *privateRepository) Marketplace() repository.MarketplaceRepository {
	return NewMarketplaceRepository(p.conn)
}

func (p *privateRepository) AdminAudit() repository.AdminAuditRepository {
	// STUB
	return nil
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
