package service

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
)

type AdminAuditService interface {
	CreateAdminAudit(audit *models.AdminAudit) error
	GetAdminAudit(id models.Snowflake) (*models.AdminAudit, error)
	DeleteAdminAudit(id models.Snowflake) error
	ListAdminAudits(opts *repository.AdminAuditListOptions) ([]*models.AdminAudit, error)
}
