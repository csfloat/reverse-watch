package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        Snowflake      `gorm:"primaryKey;not null;autoIncrement:false" json:"id"`
	CreatedAt uint64         `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt uint64         `gorm:"autoUpdateTime:milli" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"softDelete:milli" json:"deleted_at"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.CreatedAt == 0 {
		m.CreatedAt = uint64(time.Now().UnixMilli())
	}
	if m.UpdatedAt == 0 {
		m.UpdatedAt = uint64(time.Now().UnixMilli())
	}

	if !m.DeletedAt.Time.IsZero() {
		if uint64(m.DeletedAt.Time.UnixMilli())-Epoch < 0 || m.DeletedAt.Time.After(time.Now()) {
			return fmt.Errorf("deleted_at is invalid")
		}
	}

	if m.ID != 0 {
		return nil
	}

	snowflake, err := GenSnowflake()
	if err != nil {
		return err
	}
	m.ID = snowflake
	return nil
}
