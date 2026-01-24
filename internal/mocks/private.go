package mocks

import "reverse-watch/internal/domain/repository"

type MockPrivateRepository struct {
	marketplace repository.MarketplaceRepository
}

var _ repository.PrivateRepository = (*MockPrivateRepository)(nil)

func NewMockPrivateRepository() repository.PrivateRepository {
	return &MockPrivateRepository{
		marketplace: NewMockMarketplaceRepository(),
	}
}

func (m *MockPrivateRepository) Key() repository.KeyRepository {
	return nil
}

func (m *MockPrivateRepository) Marketplace() repository.MarketplaceRepository {
	return m.marketplace
}

func (m *MockPrivateRepository) AdminAudit() repository.AdminAuditRepository {
	return nil
}

func (m *MockPrivateRepository) Close() error {
	return nil
}
