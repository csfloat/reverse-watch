package testutil

import (
	"testing"

	"reverse-watch/domain/repository"

	"gorm.io/gorm"
)

type factory struct {
	db *gorm.DB

	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository
}

var _ repository.Factory = (*factory)(nil)

func NewTestFactory(t *testing.T) *factory {
	t.Helper()
	return &factory{
		db: NewTestDB(t),
	}
}

func NewTestFactoryWithDB(t *testing.T, db *gorm.DB) *factory {
	t.Helper()
	return &factory{
		db: db,
	}
}

func (f *factory) Key() repository.KeyRepository {
	return f.key
}

func (f *factory) Marketplace() repository.MarketplaceRepository {
	return f.marketplace
}

func (f *factory) AdminAudit() repository.AdminAuditRepository {
	return f.adminAudit
}

func (f *factory) Reversal() repository.ReversalRepository {
	return f.reversal
}

func (f *factory) DB() *gorm.DB {
	return f.db
}

func (f *factory) Close() error {
	return nil
}

func (f *factory) NewPrivateTransaction() repository.PrivateTransaction {
	return newPrivateTransaction(f.db.Begin(), f.key, f.marketplace, f.adminAudit)
}

func (f *factory) RunInTransactionPrivate(fn func(repository.PrivateTransaction) error) error {
	return f.db.Transaction(func(tx *gorm.DB) error {
		return fn(newPrivateTransaction(tx, f.key, f.marketplace, f.adminAudit))
	})
}

func (f *factory) NewPublicTransaction() repository.PublicTransaction {
	return newPublicTransaction(f.db.Begin(), f.reversal)
}

func (f *factory) RunInTransactionPublic(fn func(repository.PublicTransaction) error) error {
	return f.db.Transaction(func(tx *gorm.DB) error {
		return fn(newPublicTransaction(tx, f.reversal))
	})
}

func (f *factory) WithKey(key repository.KeyRepository) *factory {
	f.key = key
	return f
}

func (f *factory) WithMarketplace(marketplace repository.MarketplaceRepository) *factory {
	f.marketplace = marketplace
	return f
}

func (f *factory) WithAdminAudit(adminAudit repository.AdminAuditRepository) *factory {
	f.adminAudit = adminAudit
	return f
}

func (f *factory) WithReversal(reversal repository.ReversalRepository) *factory {
	f.reversal = reversal
	return f
}
