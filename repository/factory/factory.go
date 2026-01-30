package factory

import "reverse-watch/domain/repository"

type factory struct {
	private repository.PrivateRepository
	public  repository.PublicRepository
}

func NewFactory(private repository.PrivateRepository, public repository.PublicRepository) repository.Factory {
	return &factory{
		private: private,
		public:  public,
	}
}

func (f *factory) Key() repository.KeyRepository {
	return f.private.Key()
}

func (f *factory) Marketplace() repository.MarketplaceRepository {
	return f.private.Marketplace()
}

func (f *factory) AdminAudit() repository.AdminAuditRepository {
	return f.private.AdminAudit()
}

func (f *factory) Reversal() repository.ReversalRepository {
	return f.public.Reversal()
}
