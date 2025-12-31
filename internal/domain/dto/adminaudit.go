package dto

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/models/constants"
)

type AdminAuditListOptions struct {
	TargetActions  []constants.TargetAction
	TargetResource *models.Snowflake
}
