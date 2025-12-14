package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Reversal struct {
	Model
	SteamID         SteamID `json:"steam_id"`
	MarketplaceSlug string  `json:"marketplace_slug"`
	ReversedAt      uint64  `json:"reversed_at"`
	ExpungedAt      *uint64 `json:"expunged_at,omitempty"`
}

func (r *Reversal) BeforeCreate(tx *gorm.DB) error {
	if err := r.Model.BeforeCreate(tx); err != nil {
		return err
	}

	if !r.SteamID.IsValid() {
		return fmt.Errorf("steam_id is invalid")
	}

	if r.MarketplaceSlug == "" {
		return fmt.Errorf("marketplace_slug is required")
	}

	now := uint64(time.Now().UnixMilli())
	if r.ReversedAt > now {
		return fmt.Errorf("reversed_at cannot be in the future")
	}

	if r.ReversedAt == 0 {
		r.ReversedAt = now
	}
	return nil
}
