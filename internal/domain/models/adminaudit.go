package models

type TargetAction uint

const (
	TargetActionAddMarketplace    TargetAction = 0
	TargetActionUpdateMarketplace TargetAction = 1
	TargetActionRemoveMarketplace TargetAction = 2
	TargetActionAddKey            TargetAction = 3
	TargetActionRemoveKey         TargetAction = 4
	TargetActionUpdateReversal    TargetAction = 5
	TargetActionRemoveReversal    TargetAction = 6
	TargetActionDeleteUserData    TargetAction = 7
)

type AdminAudit struct {
	Model
	TargetAction   TargetAction `gorm:"not null" json:"target_action"`
	TargetResource *Snowflake   `json:"target_resource"`
	Details        *Jsonb       `gorm:"type:jsonb" json:"details"`
}
