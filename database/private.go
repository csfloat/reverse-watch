package database

import (
	"reverse-watch/config"
	"reverse-watch/services/private"
	"reverse-watch/types"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func InitializePrivateDB(cfg config.Config) (*gorm.DB, error) {
	conn, err := gorm.Open(sqlite.Open(cfg.PrivateDB.Filename), &gorm.Config{
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

	if err := seedScopeEnums(conn); err != nil {
		return nil, err
	}

	if err := seedMarketplaces(conn); err != nil {
		return nil, err
	}

	if err := seedAdminAPIKey(conn, cfg); err != nil {
		return nil, err
	}

	return conn, nil
}

func migratePrivateModels(tx *gorm.DB) error {
	models := []interface{}{
		(*types.ScopeEnum)(nil),
		(*types.Key)(nil),
		(*types.Marketplace)(nil),
	}

	for _, model := range models {
		if err := tx.AutoMigrate(model); err != nil {
			return err
		}
	}
	return nil
}

func seedScopeEnums(tx *gorm.DB) error {
	for scope, name := range types.ScopeToName {
		val := &types.ScopeEnum{
			Scope: scope,
			Name:  name,
		}

		if err := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(val).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedMarketplaces(tx *gorm.DB) error {
	marketplace := &types.Marketplace{
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

	id, secret, err := private.GetKeyParts(key)
	if err != nil {
		return err
	}

	adminKey := &types.Key{
		Model: types.Model{
			ID: id,
		},
		KeyHash:         private.HashSecret(secret, salt),
		Salt:            salt,
		MarketplaceSlug: "csfloat",
		Scope:           types.ScopeAdmin,
	}

	return tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Create(adminKey).Error
}
