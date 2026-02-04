package testutil

import (
	"testing"

	"reverse-watch/domain/repository"
)

type factory struct {
	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository
}

func NewTestFactory(t *testing.T) *factory {
	t.Helper()
	return &factory{}
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

func (f *factory) Close() error {
	return nil
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
