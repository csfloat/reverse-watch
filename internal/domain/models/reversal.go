package models

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Reversal struct {
	Model
	SteamID         SteamID  `json:"steam_id"`
	MarketplaceSlug string   `json:"marketplace_slug"`
	Source          *Source  `json:"source,omitempty"`
	RelatedSteamID  *SteamID `json:"related_steam_id,omitempty"`
	ReversedAt      uint64   `json:"reversed_at"`
	ExpungedAt      *uint64  `json:"expunged_at,omitempty"`
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

	if err := ValidateSourceAndRelatedID(r.Source, r.RelatedSteamID); err != nil {
		return err
	}

	now := uint64(time.Now().UnixMilli())
	if r.ReversedAt > now {
		return fmt.Errorf("reversed_at cannot be in the future")
	}

	if r.ExpungedAt != nil && *r.ExpungedAt > now {
		return fmt.Errorf("expunged_at cannot be in the future")
	}

	if r.ReversedAt == 0 {
		r.ReversedAt = now
	}
	return nil
}

func ValidateSourceAndRelatedID(source *Source, relatedSteamID *SteamID) error {
	if relatedSteamID != nil {
		if !relatedSteamID.IsValid() {
			return fmt.Errorf("related_steam_id is invalid")
		}
		if source == nil || *source != SourceRelatedUser {
			return fmt.Errorf("invalid related_steam_id and source combination")
		}
	}

	if source != nil {
		if *source == SourceRelatedUser {
			if relatedSteamID == nil || !relatedSteamID.IsValid() {
				return fmt.Errorf("invalid related_steam_id and source combination")
			}
		} else if relatedSteamID != nil {
			return fmt.Errorf("invalid related_steam_id and source combination")
		}
	}
	return nil
}

type Source uint

const (
	SourceDirect      Source = 0
	SourceRelatedUser Source = 1
	SourceUserReport  Source = 2
)

func (s *Source) String() string {
	var source string
	switch *s {
	case SourceDirect:
		source = "direct"
	case SourceRelatedUser:
		source = "related_user"
	case SourceUserReport:
		source = "user_report"
	}
	return source
}

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
