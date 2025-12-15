package private

import (
	"path/filepath"

	"reverse-watch/internal/config"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/pkg/crypto"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type privateRepository struct {
	conn *gorm.DB
}

var _ repository.PrivateRepository = (*privateRepository)(nil)

func NewPrivateRepository(cfg config.Config) (repository.PrivateRepository, error) {
	rootDir, err := config.GetProjectRootDir()
	if err != nil {
		return nil, err
	}

	dsn := filepath.Join(rootDir, cfg.DataDir, cfg.PrivateDB.Filename)
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

	if err := seedAdminAPIKey(conn, cfg); err != nil {
		return nil, err
	}

	return &privateRepository{
		conn: conn,
	}, nil
}

func (p *privateRepository) Key() repository.KeyRepository {
	return NewKeyRepository(p.conn)
}

func (p *privateRepository) Marketplace() repository.MarketplaceRepository {
	return NewMarketplaceRepository(p.conn)
}

func (p *privateRepository) AdminAudit() repository.AdminAuditRepository {
	return NewAdminAuditRepository(p.conn)
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

func seedAdminAPIKey(tx *gorm.DB, cfg config.Config) error {
	key := cfg.Admin.APIKey
	if key == "" {
		return nil
	}
	salt := cfg.Admin.Salt
	if salt == "" {
		return nil
	}

	id, secret, err := crypto.ParseSecretKey(key)
	if err != nil {
		return err
	}

	permissions := models.PermissionAdmin
	permissions.AddPermission(models.PermissionDelete)
	permissions.AddPermission(models.PermissionManage)
	permissions.AddPermission(models.PermissionWrite)
	permissions.AddPermission(models.PermissionRead)

	adminKey := &models.Key{
		Model: models.Model{
			ID: models.Snowflake(id),
		},
		KeyHash:         crypto.HashSecret(secret, salt),
		Salt:            salt,
		MarketplaceSlug: "csfloat",
		Permissions:     permissions,
	}

	return tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(adminKey).Error
}
