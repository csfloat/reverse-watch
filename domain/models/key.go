package models

import (
	"fmt"

	"reverse-watch/domain/models/constants"

	"gorm.io/gorm"
)

type Key struct {
	// ID is the hash of the secret key
	ID              string                `gorm:"primaryKey" json:"id"`
	CreatedAt       uint64                `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt       uint64                `gorm:"autoUpdateTime:milli" json:"updated_at"`
	Environment     constants.Environment `json:"environment"`
	MarketplaceSlug string                `json:"marketplace_slug"`
	Marketplace     *Marketplace          `gorm:"foreignKey:MarketplaceSlug;references:Slug" json:"-"`
	Permissions     Permissions           `json:"permissions"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
		return fmt.Errorf("id is required")
	}
	if k.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	if k.MarketplaceSlug == "" {
		return fmt.Errorf("marketplace_slug is required")
	}
	if k.Permissions == PermissionNone {
		return fmt.Errorf("at least one permission is required")
	}

	if k.HasPermissions(PermissionAdmin) && !k.IsOwnedByCSFloat() {
		return fmt.Errorf("admin scoped keys can only be created for CSFloat")
	}
	return nil
}

func (k *Key) HasPermissions(permissions ...Permissions) bool {
	return k.Permissions.HasPermissions(permissions...)
}

func (k *Key) IsOwnedByCSFloat() bool {
	return k.MarketplaceSlug == "csfloat"
}
