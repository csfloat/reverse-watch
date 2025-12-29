package models

import (
	"fmt"

	"reverse-watch/internal/config"

	"gorm.io/gorm"
)

type Key struct {
	KeyHash         string             `gorm:"primaryKey" json:"-"`
	CreatedAt       uint64             `gorm:"autoCreateTime:milli" json:"created_at"`
	Environment     config.Environment `json:"-"`
	MarketplaceSlug string             `json:"marketplace_slug"`
	Marketplace     *Marketplace       `json:"-"`
	Permissions     Permissions        `json:"permissions"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
	if k.KeyHash == "" {
		return fmt.Errorf("key hash is required")
	}
	if k.Environment == "" {
		return fmt.Errorf("environment is required")
	}

	if k.HasPermissions(PermissionAdmin) && k.MarketplaceSlug != "csfloat" {
		return fmt.Errorf("admin scoped keys can only be created for CSFloat")
	}
	return nil
}

func (k *Key) HasPermissions(permissions ...Permissions) bool {
	return k.Permissions.HasPermissions(permissions...)
}
