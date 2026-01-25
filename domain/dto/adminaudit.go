package dto

import (
	"reverse-watch/domain/models"
)

type AdminAuditListOptions struct {
	TargetActions      []models.TargetAction
	TargetResourceType *models.TargetResourceType
	TargetResource     *string
}
