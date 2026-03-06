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
	// The timestamp of the reversal in milliseconds since the Unix epoch
	ReversedAt         uint64 `json:"reversed_at"`
	ReporterInternalID *uint  `json:"-"`
	// The timestamp the reversal was expunged at in milliseconds since the Unix epoch
	ExpungedAt *uint64 `json:"expunged_at,omitempty"`
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

	if r.ExpungedAt != nil {
		if *r.ExpungedAt == 0 || *r.ExpungedAt < r.CreatedAt || *r.ExpungedAt > now {
			return fmt.Errorf("expunged_at is invalid")
		}
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

	if source != nil && *source == SourceRelatedUser {
		if relatedSteamID == nil {
			return fmt.Errorf("related_steam_id is required when source is %q", "related_user")
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
	if s == nil {
		return ""
	}

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
	source := s.String()
	if source == "" {
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
