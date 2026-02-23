package dto

import (
	"reverse-watch/domain/models"
)

type AdminAuditListOptions struct {
	TargetActions      []models.TargetAction
	TargetResourceType *models.TargetResourceType
	TargetResource     *string
}

type KeyAuditDetails struct {
	MarketplaceSlug string             `json:"marketplace_slug"`
	Permissions     models.Permissions `json:"permissions"`
	AdminKey        string             `json:"admin_key"`
}
