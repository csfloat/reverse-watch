package models

import (
	"fmt"

	"gorm.io/gorm"
)

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
)

type AdminAudit struct {
	Model
	TargetAction       TargetAction       `json:"target_action"`
	TargetResourceType TargetResourceType `json:"target_resource_type"`
	TargetResource     string             `json:"target_resource"`
	InitiatorKey       string             `json:"initiator_key"`
	Details            *RawJsonb          `json:"details,omitempty"`
}

func (a *AdminAudit) BeforeCreate(tx *gorm.DB) error {
	if err := a.Model.BeforeCreate(tx); err != nil {
		return err
	}

	if a.InitiatorKey == "" {
		return fmt.Errorf("initiator_key is required")
	}
	return nil
}

func NewMarketplaceAdminAudit(action TargetAction, slug, initiatorKey string, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeMarketplace,
		TargetResource:     slug,
		InitiatorKey:       initiatorKey,
		Details:            details,
	}
}

func NewKeyAdminAudit(action TargetAction, id, initiatorKey string, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeKey,
		TargetResource:     id,
		InitiatorKey:       initiatorKey,
		Details:            details,
	}
}

func NewReversalAdminAudit(action TargetAction, id Snowflake, initiatorKey string, details *RawJsonb) *AdminAudit {
	return &AdminAudit{
		TargetAction:       action,
		TargetResourceType: TargetResourceTypeReversal,
		TargetResource:     id.String(),
		InitiatorKey:       initiatorKey,
		Details:            details,
	}
}
