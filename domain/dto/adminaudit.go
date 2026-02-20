package dto

import (
	"reverse-watch/domain/models"
)

type AdminAuditListOptions struct {
	TargetActions      []models.TargetAction
	TargetResourceType *models.TargetResourceType
	TargetResource     *string
}

type DeleteKeyDetails struct {
	MarketplaceSlug string             `json:"marketplace_slug"`
	Permissions     models.Permissions `json:"permissions"`
}
