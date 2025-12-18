package adminaudit

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
)

type adminAuditService struct {
	repository.PrivateRepository
}

var _ service.AdminAuditService = (*adminAuditService)(nil)

func NewAdminAuditService(repo repository.PrivateRepository) service.AdminAuditService {
	return &adminAuditService{
		PrivateRepository: repo,
	}
}

func (a *adminAuditService) CreateAdminAudit(audit *models.AdminAudit) error {
	return a.AdminAudit().Create(audit)
}

func (a *adminAuditService) GetAdminAudit(id models.Snowflake) (*models.AdminAudit, error) {
	return a.AdminAudit().Read(id)
}

func (a *adminAuditService) DeleteAdminAudit(id models.Snowflake) error {
	return a.AdminAudit().Delete(id)
}

func (a *adminAuditService) ListAdminAudits(opts *repository.AdminAuditListOptions) ([]*models.AdminAudit, error) {
	return a.AdminAudit().List(opts)
}
