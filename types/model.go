package types

import "gorm.io/gorm"

type Model struct {
	ID        Snowflake `gorm:"primaryKey;autoIncrement:false" json:"id"`
	CreatedAt uint64    `gorm:"autoCreateTime:milli" json:"created_at"`
	UpdatedAt uint64    `gorm:"autoUpdateTime:milli" json:"updated_at"`
}

func (m *Model) BeforeCreate(tx *gorm.DB) error {
	snowflake, err := GenSnowflake()
	if err != nil {
		return err
	}
	m.ID = snowflake
	return nil
}
