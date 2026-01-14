package private

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"

	"gorm.io/gorm"
)

type adminAuditRepository struct {
	conn *gorm.DB
}

var _ repository.AdminAuditRepository = (*adminAuditRepository)(nil)

func NewAdminAuditRepository(conn *gorm.DB) repository.AdminAuditRepository {
	return &adminAuditRepository{
		conn: conn,
	}
}

func (a *adminAuditRepository) Create(audit *models.AdminAudit) error {
	return a.conn.Model(&models.AdminAudit{}).Create(audit).Error
}

func (a *adminAuditRepository) Read(id models.Snowflake) (*models.AdminAudit, error) {
	var audit *models.AdminAudit
	if err := a.conn.Model(&models.AdminAudit{}).Where("id = ?", id).First(&audit).Error; err != nil {
		return nil, err
	}
	return audit, nil
}

func (a *adminAuditRepository) Delete(id models.Snowflake) error {
	tx := a.conn.Where("id = ?", id).Delete(&models.AdminAudit{})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (a *adminAuditRepository) buildListQuery(opts *dto.AdminAuditListOptions) *gorm.DB {
	query := a.conn.Model(&models.AdminAudit{})
	if opts == nil {
		return query
	}
	if opts.TargetResourceType != nil {
		query = query.Where("target_resource_type = ?", opts.TargetResourceType)
	}
	if opts.TargetResource != nil {
		query = query.Where("target_resource = ?", opts.TargetResource)
	}
	if len(opts.TargetActions) > 0 {
		query = query.Where("target_action IN (?)", opts.TargetActions)
	}
	return query
}

func (a *adminAuditRepository) List(opts *dto.AdminAuditListOptions) ([]*models.AdminAudit, error) {
	query := a.buildListQuery(opts)

	var audits []*models.AdminAudit
	if err := query.Order("id DESC").Find(&audits).Error; err != nil {
		return nil, err
	}
	return audits, nil
}
