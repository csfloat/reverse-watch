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
	Details            *RawJsonb          `gorm:"type:jsonb" json:"details"`
}

func NewMarketplaceAdminAudit(action TargetAction, slug string, details interface{}) (*AdminAudit, error) {
	rawDetails, err := toRawJsonb(details)
	if err != nil {
		return nil, err
	}
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeMarketplace,
		TargetResource:     slug,
		Details:            rawDetails,
	}, nil
}

func NewKeyAdminAudit(action TargetAction, id string, details interface{}) (*AdminAudit, error) {
	rawDetails, err := toRawJsonb(details)
	if err != nil {
		return nil, err
	}
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeKey,
		TargetResource:     id,
		Details:            rawDetails,
	}, nil
}

func NewReversalAdminAudit(action TargetAction, id Snowflake, details interface{}) (*AdminAudit, error) {
	rawDetails, err := toRawJsonb(details)
	if err != nil {
		return nil, err
	}
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeReversal,
		TargetResource:     id.String(),
		Details:            rawDetails,
	}, nil
}
