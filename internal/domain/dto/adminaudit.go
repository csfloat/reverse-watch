package dto

import (
	"reverse-watch/internal/domain/models/types"
)

type AdminAuditListOptions struct {
	TargetActions  []types.TargetAction
	TargetResource *types.Snowflake
}
