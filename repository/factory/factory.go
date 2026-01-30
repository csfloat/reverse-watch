package factory

import "reverse-watch/domain/repository"

type factory struct {
	privateRepo repository.PrivateRepository
	publicRepo  repository.PublicRepository
}

func NewFactory(privateRepo repository.PrivateRepository, publicRepo repository.PublicRepository) repository.RepositoryFactory {
	return &factory{
		privateRepo: privateRepo,
		publicRepo:  publicRepo,
	}
}

func (f *factory) Private() repository.PrivateRepository {
	return f.privateRepo
}

func (f *factory) Public() repository.PublicRepository {
	return f.publicRepo
}
