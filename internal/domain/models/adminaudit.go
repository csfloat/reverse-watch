package models

import "reverse-watch/internal/domain/models/constants"

type AdminAudit struct {
	Model
	TargetAction   constants.TargetAction `gorm:"not null" json:"target_action"`
	TargetResource *Snowflake             `json:"target_resource"`
	Details        *Jsonb                 `gorm:"type:jsonb" json:"details"`
}
