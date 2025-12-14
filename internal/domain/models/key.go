package models

import (
	"fmt"

	"gorm.io/gorm"
)

type Key struct {
	Model
	KeyHash         string       `gorm:"unique" json:"-"`
	Salt            string       `gorm:"unique" json:"-"`
	MarketplaceSlug string       `json:"marketplace_slug"`
	Marketplace     *Marketplace `json:"-"`
	Permissions     Permissions  `json:"permissions"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
	if err := k.Model.BeforeCreate(tx); err != nil {
		return err
	}

	if k.KeyHash == "" {
		return fmt.Errorf("key hash is required")
	}

	if k.HasPermissions(PermissionAdmin) && k.MarketplaceSlug != "csfloat" {
		return fmt.Errorf("admin scoped keys can only be created for CSFloat")
	}
	return nil
}

func (k *Key) HasPermissions(permissions ...Permissions) bool {
	return k.Permissions.HasPermissions(permissions...)
}
