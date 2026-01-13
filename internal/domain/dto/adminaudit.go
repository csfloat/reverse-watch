package dto

import (
	"reverse-watch/internal/domain/models"
)

type AdminAuditListOptions struct {
	TargetActions  []models.TargetAction
	TargetResource *string
}

type MarketplaceAuditDetails struct {
	MarketplaceUpdateOptions
}
