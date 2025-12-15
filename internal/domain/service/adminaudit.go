package service

import "reverse-watch/internal/domain/models"

type AdminAuditService interface {
	CreateAdminAudit(audit *models.AdminAudit) error
	GetAdminAudit(id models.Snowflake) (*models.AdminAudit, error)
	DeleteAdminAudit(id models.Snowflake) error
	ListAdminAudits(opts AdminAuditListOptions) ([]*models.AdminAudit, error)
}

type AdminAuditListOptions struct {
	TargetActions  []models.TargetAction
	TargetResource *models.Snowflake
}
