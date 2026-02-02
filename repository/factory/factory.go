package factory

import (
	"reverse-watch/domain/repository"
	"reverse-watch/domain/secret"
	"reverse-watch/repository/private"
	"reverse-watch/repository/public"

	"gorm.io/gorm"
)

type factory struct {
	private *gorm.DB
	public  *gorm.DB
	keygen  secret.KeyGenerator

	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository
}

func NewFactory(private, public *gorm.DB, keygen secret.KeyGenerator) repository.Factory {
	return &factory{
		private: private,
		public:  public,
		keygen:  keygen,
	}
}

func (f *factory) Key() repository.KeyRepository {
	if f.key == nil {
		f.key = private.NewKeyRepository(f.private, f.keygen)
	}
	return f.key
}

func (f *factory) Marketplace() repository.MarketplaceRepository {
	if f.marketplace == nil {
		f.marketplace = private.NewMarketplaceRepository(f.private)
	}
	return f.marketplace
}

func (f *factory) AdminAudit() repository.AdminAuditRepository {
	if f.adminAudit == nil {
		f.adminAudit = private.NewAdminAuditRepository(f.private)
	}
	return f.adminAudit
}

func (f *factory) Reversal() repository.ReversalRepository {
	if f.reversal == nil {
		f.reversal = public.NewReversalRepository(f.public)
	}
	return f.reversal
}
