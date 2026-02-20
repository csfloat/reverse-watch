package models

type TargetAction uint

const (
	TargetActionUnknown           TargetAction = 0
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
	TargetResourceTypeUnknown     TargetResourceType = 0
	TargetResourceTypeMarketplace TargetResourceType = 1
	TargetResourceTypeKey         TargetResourceType = 2
	TargetResourceTypeReversal    TargetResourceType = 3
	TargetResourceTypeUser        TargetResourceType = 4
)

type AdminAudit struct {
	Model
	TargetAction       TargetAction       `json:"target_action"`
	TargetResourceType TargetResourceType `json:"target_resource_type"`
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

func NewUserAdminAudit(action TargetAction, steamId SteamID, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeUser,
		TargetResource:     steamId.String(),
		Details:            details,
	}
}
