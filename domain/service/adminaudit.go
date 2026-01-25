package service

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
)

type AdminAuditService interface {
	CreateAdminAudit(audit *models.AdminAudit) error
	GetAdminAudit(id models.Snowflake) (*models.AdminAudit, error)
	DeleteAdminAudit(id models.Snowflake) error
	ListAdminAudits(opts *dto.AdminAuditListOptions) ([]*models.AdminAudit, error)
}
