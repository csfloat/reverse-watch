package types

import (
	"fmt"

	"gorm.io/gorm"
)

type Key struct {
	Model
	KeyHash         string       `gorm:"unique" json:"-"`
	Salt            string       `json:"salt"`
	MarketplaceSlug string       `json:"marketplace_slug"`
	Marketplace     *Marketplace `json:"-"`
	Scope           Scope        `json:"scope"`
	ScopeDetail     *ScopeEnum   `gorm:"foreignKey:Scope;references:Scope" json:"-"`
}

func (k *Key) BeforeCreate(tx *gorm.DB) error {
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
