package models

import (
	"encoding/json"
	"fmt"
	"time"

	"reverse-watch/internal/domain/models/types"

	"gorm.io/gorm"
)

type Reversal struct {
	types.Model
	SteamID         types.SteamID  `json:"steam_id"`
	MarketplaceSlug string         `json:"marketplace_slug"`
	Source          *Source        `json:"source,omitempty"`
	RelatedSteamID  *types.SteamID `json:"related_steam_id,omitempty"`
	ReversedAt      uint64         `json:"reversed_at"`
	ExpungedAt      *uint64        `json:"expunged_at,omitempty"`
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

	if r.RelatedSteamID != nil {
		if !r.RelatedSteamID.IsValid() {
			return fmt.Errorf("related_steam_id is invalid")
		}
		if r.Source == nil || *r.Source != SourceRelatedUser {
			return fmt.Errorf("invalid related_steam_id and source combination")
		}
	}

	if r.Source != nil && *r.Source == SourceRelatedUser {
		if r.RelatedSteamID == nil || !r.RelatedSteamID.IsValid() {
			return fmt.Errorf("invalid related_steam_id and source combination")
		}
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

type Source uint

const (
	SourceDirect      Source = 0
	SourceRelatedUser Source = 1
	SourceUserReport  Source = 2
)

func (s *Source) MarshalJSON() ([]byte, error) {
	var source string
	switch *s {
	case SourceDirect:
		source = "direct"
	case SourceRelatedUser:
		source = "related_user"
	case SourceUserReport:
		source = "user_report"
	default:
		return nil, fmt.Errorf("invalid source value")
	}
	return json.Marshal(source)
}

func (s *Source) UnmarshalJSON(b []byte) error {
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	switch str {
	case "direct":
		*s = SourceDirect
	case "related_user":
		*s = SourceRelatedUser
	case "user_report":
		*s = SourceUserReport
	default:
		return fmt.Errorf("invalid source value")
	}
	return nil
}
