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

type TargetResourceType uint

const (
	TargetResourceTypeMarketplace TargetResourceType = 0
	TargetResourceTypeKey         TargetResourceType = 1
	TargetResourceTypeReversal    TargetResourceType = 2
)

type AdminAudit struct {
	Model
	TargetAction       TargetAction       `json:"target_action"`
	TargetResourceType TargetResourceType `json:"resource_type"`
	TargetResource     string             `json:"target_resource"`
	Details            *Jsonb             `gorm:"type:jsonb" json:"details"`
}

func NewMarketplaceAdminAudit(action TargetAction, slug string, details *Jsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeMarketplace,
		TargetResource:     slug,
		Details:            details,
	}
}

func NewKeyAdminAudit(action TargetAction, id string, details *Jsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeKey,
		TargetResource:     id,
		Details:            details,
	}
}

func NewReversalAdminAudit(action TargetAction, id Snowflake, details *Jsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeReversal,
		TargetResource:     id.String(),
		Details:            details,
	}
}
