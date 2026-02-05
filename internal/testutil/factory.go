package testutil

import (
	"testing"

	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/repository"
	isecret "reverse-watch/domain/secret"
	"reverse-watch/secret"

	"gorm.io/gorm"
)

type factory struct {
	db     *gorm.DB
	keygen isecret.KeyGenerator

	// For non-transactional access
	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository

	// Constructor functions for creating transactional repositories
	keyConstructor         func(*gorm.DB, isecret.KeyGenerator) repository.KeyRepository
	marketplaceConstructor func(*gorm.DB) repository.MarketplaceRepository
	adminAuditConstructor  func(*gorm.DB) repository.AdminAuditRepository
	reversalConstructor    func(*gorm.DB) repository.ReversalRepository
}

var _ repository.Factory = (*factory)(nil)

func NewTestFactory(t *testing.T) *factory {
	t.Helper()
	return &factory{
		db:     NewTestDB(t),
		keygen: secret.NewKeyGenerator(constants.EnvironmentDevelopment),
	}
}

func NewTestFactoryWithDB(t *testing.T, db *gorm.DB) *factory {
	t.Helper()
	return &factory{
		db:     db,
		keygen: secret.NewKeyGenerator(constants.EnvironmentDevelopment),
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
	return newPrivateTransaction(f.db.Begin(), f.keyConstructor(f.db, f.keygen), f.marketplaceConstructor(f.db), f.adminAuditConstructor(f.db))
}

func (f *factory) RunInTransactionPrivate(fn func(repository.PrivateTransaction) error) error {
	return f.db.Transaction(func(gormTx *gorm.DB) error {
		tx := newPrivateTransaction(gormTx, f.keyConstructor(gormTx, f.keygen), f.marketplaceConstructor(gormTx), f.adminAuditConstructor(gormTx))
		return fn(tx)
	})
}

func (f *factory) NewPublicTransaction() repository.PublicTransaction {
	return newPublicTransaction(f.db.Begin(), f.reversalConstructor(f.db))
}

func (f *factory) RunInTransactionPublic(fn func(repository.PublicTransaction) error) error {
	return f.db.Transaction(func(gormTx *gorm.DB) error {
		tx := newPublicTransaction(gormTx, f.reversalConstructor(gormTx))
		return fn(tx)
	})
}

func (f *factory) WithKey(constructor func(*gorm.DB, isecret.KeyGenerator) repository.KeyRepository) *factory {
	f.keyConstructor = constructor
	f.key = constructor(f.db, f.keygen)
	return f
}

func (f *factory) WithMarketplace(constructor func(*gorm.DB) repository.MarketplaceRepository) *factory {
	f.marketplaceConstructor = constructor
	f.marketplace = constructor(f.db)
	return f
}

func (f *factory) WithAdminAudit(constructor func(*gorm.DB) repository.AdminAuditRepository) *factory {
	f.adminAuditConstructor = constructor
	f.adminAudit = constructor(f.db)
	return f
}

func (f *factory) WithReversal(constructor func(*gorm.DB) repository.ReversalRepository) *factory {
	f.reversalConstructor = constructor
	f.reversal = constructor(f.db)
	return f
}
