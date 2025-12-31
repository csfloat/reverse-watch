package models

import "reverse-watch/internal/domain/models/types"

type AdminAudit struct {
	types.Model
	TargetAction   types.TargetAction `gorm:"not null" json:"target_action"`
	TargetResource *types.Snowflake   `json:"target_resource"`
	Details        *types.Jsonb       `gorm:"type:jsonb" json:"details"`
}
