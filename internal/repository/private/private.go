package private

import (
	"path/filepath"

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

type privateRepository struct {
	conn *gorm.DB
}

var _ repository.PrivateRepository = (*privateRepository)(nil)

func NewPrivateRepository(cfg config.Config, keygen secret.KeyGenerator) (repository.PrivateRepository, error) {
	rootDir, err := config.GetProjectRootDir()
	if err != nil {
		return nil, err
	}

	dsn := filepath.Join(rootDir, cfg.StaticDir, "private.db")
	conn, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	if err := conn.Exec("PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL").Error; err != nil {
		return nil, err
	}

	if err := migratePrivateModels(conn); err != nil {
		return nil, err
	}

	if err := seedMarketplaces(conn); err != nil {
		return nil, err
	}

	if err := seedAdminAPIKey(conn, keygen); err != nil {
		return nil, err
	}

	return &privateRepository{
		conn: conn,
	}, nil
}

func (p *privateRepository) Key() repository.KeyRepository {
	return nil
}

func (p *privateRepository) Marketplace() repository.MarketplaceRepository {
	return NewMarketplaceRepository(p.conn)
}

func (p *privateRepository) AdminAudit() repository.AdminAuditRepository {
	return nil
}

func migratePrivateModels(tx *gorm.DB) error {
	privateModels := []interface{}{
		(*models.Key)(nil),
		(*models.Marketplace)(nil),
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
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM keys WHERE permissions & ? = ?)`, models.PermissionAdmin, models.PermissionAdmin).
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

	logging.Log.Infof("GENERATED ADMIN SECRET KEY: %s\n\nSAVE THIS FOR FUTURE PURPOSES, WON'T BE SHOWN AGAIN!", secretKey.Format())

	permissions := models.PermissionAdmin
	permissions.AddAllPermissions()

	adminKey := &models.Key{
		KeyHash:         secretKey.Hash(),
		MarketplaceSlug: "csfloat",
		Permissions:     permissions,
	}

	return tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(adminKey).Error
}
