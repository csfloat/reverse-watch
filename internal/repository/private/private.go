package private

import (
	"encoding/base64"
	"path/filepath"

	"reverse-watch/internal/config"
	"reverse-watch/internal/crypto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
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

func NewPrivateRepository(cfg config.Config) (repository.PrivateRepository, error) {
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

	if err := seedAdminAPIKey(conn); err != nil {
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
	return nil
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

func seedAdminAPIKey(tx *gorm.DB) error {
	var exists bool
	err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM keys WHERE permissions & ? = ?)`, models.PermissionAdmin, models.PermissionAdmin).
		Row().Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	id, secret, salt, err := crypto.GenerateSecretKey()
	if err != nil {
		return err
	}

	encodedSecret := base64.RawURLEncoding.EncodeToString(secret)
	encodedSalt := base64.RawURLEncoding.EncodeToString(salt)

	secretKey := crypto.FormatAPIKey(uint64(id), encodedSecret)
	logging.Log.Infof("Generated admin secret key: %s", secretKey)

	permissions := models.PermissionAdmin
	permissions.AddPermission(models.PermissionDelete)
	permissions.AddPermission(models.PermissionManage)
	permissions.AddPermission(models.PermissionWrite)
	permissions.AddPermission(models.PermissionRead)
	permissions.AddPermission(models.PermissionExport)

	adminKey := &models.Key{
		Model: models.Model{
			ID: id,
		},
		KeyHash:         crypto.HashSecret(encodedSecret, encodedSalt),
		Salt:            encodedSalt,
		MarketplaceSlug: "csfloat",
		Permissions:     permissions,
	}

	return tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(adminKey).Error
}
