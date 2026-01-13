package models

type TargetAction uint

const (
	TargetActionNone              TargetAction = 0
	TargetActionAddMarketplace    TargetAction = 1
	TargetActionUpdateMarketplace TargetAction = 2
	TargetActionRemoveMarketplace TargetAction = 3
	TargetActionAddKey            TargetAction = 4
	TargetActionRemoveKey         TargetAction = 5
	TargetActionUpdateReversal    TargetAction = 6
	TargetActionRemoveReversal    TargetAction = 7
	TargetActionDeleteUserData    TargetAction = 8
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
	Details            *RawJsonb          `json:"details"`
}

func NewMarketplaceAdminAudit(action TargetAction, slug string, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeMarketplace,
		TargetResource:     slug,
		Details:            details,
	}
}

func NewKeyAdminAudit(action TargetAction, id string, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeKey,
		TargetResource:     id,
		Details:            details,
	}
}

func NewReversalAdminAudit(action TargetAction, id Snowflake, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeReversal,
		TargetResource:     id.String(),
		Details:            details,
	}
}
