package types

import (
	"fmt"

	"gorm.io/gorm"
)

type Marketplace struct {
	Slug      string `gorm:"primaryKey" json:"slug"`
	CreatedAt int64  `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt int64  `gorm:"autoUpdateTime:milli" json:"updated_at"`
	Name      string `json:"name"`
	IsActive  bool   `json:"is_active"`
}

func (m *Marketplace) BeforeCreate(tx *gorm.DB) error {
	if m.Slug == "" {
		return fmt.Errorf("slug is required")
	}
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}
