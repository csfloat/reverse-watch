package models

import (
	"fmt"

	"gorm.io/gorm"
)

type Key struct {
	// ID is the hash of the secret key
	ID              string       `gorm:"primaryKey" json:"-"`
	CreatedAt       uint64       `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt       uint64       `gorm:"autoUpdateTime:milli" json:"updated_at"`
	Environment     Environment  `json:"-"`
	MarketplaceSlug string       `json:"marketplace_slug"`
	Marketplace     *Marketplace `json:"-"`
	Permissions     Permissions  `json:"permissions"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
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
