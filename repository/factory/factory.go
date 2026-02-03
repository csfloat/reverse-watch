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

	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
	reversal    repository.ReversalRepository
}

func NewFactory(privateDB, publicDB *gorm.DB, keygen secret.KeyGenerator) repository.Factory {
	return &factory{
		private:     privateDB,
		public:      publicDB,
		key:         private.NewKeyRepository(privateDB, keygen),
		marketplace: private.NewMarketplaceRepository(privateDB),
		adminAudit:  private.NewAdminAuditRepository(privateDB),
		reversal:    public.NewReversalRepository(publicDB),
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
