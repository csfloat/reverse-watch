package types

import (
	"fmt"

	"gorm.io/gorm"
)

type Key struct {
	// ID is the Public UUID that is used to identify the key
	ID              string       `gorm:"primaryKey" json:"id"`
	CreatedAt       int64        `gorm:"autoCreateTime:milli" json:"created_at"`
	KeyHash         string       `gorm:"unique" json:"key_hash"`
	MarketplaceSlug string       `json:"marketplace_slug"`
	Marketplace     *Marketplace `json:"marketplace"`
	Scope           Scope        `json:"scope"`
	ScopeDetail     *ScopeEnum   `gorm:"foreignKey:Scope;references:Scope" json:"-"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
	if k.ID == "" {
		return fmt.Errorf("id is required")
	}
	if k.KeyHash == "" {
		return fmt.Errorf("key hash is required")
	}
	if !k.Scope.IsValid() {
		return fmt.Errorf("invalid scope")
	}
	if k.Scope == ScopeAdmin && k.MarketplaceSlug != "csfloat" {
		return fmt.Errorf("admin scoped keys can only be created by CSFloat")
	}
	return nil
}
