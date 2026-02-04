package factory

import (
	"reverse-watch/domain/repository"

	"gorm.io/gorm"
)

type privateTransaction struct {
	tx          *gorm.DB
	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
}

func newPrivateTransaction(tx *gorm.DB, key repository.KeyRepository, marketplace repository.MarketplaceRepository, adminAudit repository.AdminAuditRepository) *privateTransaction {
	return &privateTransaction{
		tx:          tx,
		key:         key,
		marketplace: marketplace,
		adminAudit:  adminAudit,
	}
}

func (t *privateTransaction) Key() repository.KeyRepository {
	return t.key
}

func (t *privateTransaction) Marketplace() repository.MarketplaceRepository {
	return t.marketplace
}

func (t *privateTransaction) AdminAudit() repository.AdminAuditRepository {
	return t.adminAudit
}

func (t *privateTransaction) Commit() error {
	return t.tx.Commit().Error
}

func (t *privateTransaction) Rollback() error {
	return t.tx.Rollback().Error
}

type publicTransaction struct {
	tx       *gorm.DB
	reversal repository.ReversalRepository
}

func newPublicTransaction(tx *gorm.DB, reversal repository.ReversalRepository) *publicTransaction {
	return &publicTransaction{
		tx:       tx,
		reversal: reversal,
	}
}

func (t *publicTransaction) Reversal() repository.ReversalRepository {
	return t.reversal
}

func (t *publicTransaction) Commit() error {
	return t.tx.Commit().Error
}

func (t *publicTransaction) Rollback() error {
	return t.tx.Rollback().Error
}
