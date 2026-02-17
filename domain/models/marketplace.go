package models

import (
	"fmt"
	"regexp"

	"gorm.io/gorm"
)

type Marketplace struct {
	Slug      string         `gorm:"primaryKey" json:"slug"`
	CreatedAt uint64         `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt uint64         `gorm:"autoUpdateTime:milli" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"softDelete:milli" json:"deleted_at"`
	Name      string         `json:"name"`
	IsActive  bool           `json:"is_active"`
}

func (m *Marketplace) BeforeCreate(tx *gorm.DB) error {
	return m.Validate()
}

func (m *Marketplace) Validate() error {
	if len(m.Slug) == 0 || len(m.Slug) > 25 {
		return fmt.Errorf("slug must be between 1 and 25 characters long")
	}
	if !regexp.MustCompile(`^[a-zA-Z0-9-]+$`).MatchString(m.Slug) {
		return fmt.Errorf("slug must contain only letters, numbers, and hyphens")
	}

	if len(m.Name) == 0 || len(m.Name) > 50 {
		return fmt.Errorf("name must be between 1 and 50 characters long")
	}
	return nil
}
