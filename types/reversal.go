package types

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Reversal struct {
	ID              uint    `gorm:"primaryKey" json:"id"`
	CreatedAt       int64   `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt       int64   `gorm:"autoUpdateTime:milli" json:"updated_at"`
	SteamID         SteamID `json:"steam_id"`
	MarketplaceSlug string  `json:"marketplace_slug"`
	ReversedAt      int64   `json:"reversed_at"`
	ExpungedAt      int64   `json:"expunged_at"`
}

func (r *Reversal) BeforeCreate(tx *gorm.DB) error {
	if !r.SteamID.IsValid() {
		return fmt.Errorf("steam_id is invalid")
	}

	if r.MarketplaceSlug == "" {
		return fmt.Errorf("marketplace_slug is required")
	}

	now := time.Now().UnixMilli()
	if r.ReversedAt > now {
		return fmt.Errorf("reversed_at cannot be in the future")
	}

	if r.ReversedAt <= 0 {
		r.ReversedAt = now
	}
	return nil
}
