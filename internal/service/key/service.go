package key

import (
	"crypto/subtle"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/secret"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"
)

type keyService struct {
	repository.PrivateRepository
	secret.KeyGenerator
}

var _ service.KeyService = (*keyService)(nil)

func NewKeyService(repo repository.PrivateRepository, keygen secret.KeyGenerator) service.KeyService {
	return &keyService{
		PrivateRepository: repo,
		KeyGenerator:      keygen,
	}
}

func (s *keyService) CreateKey(marketplaceSlug string, permissions models.Permissions) (*dto.RawKey, error) {
	secretKey, err := s.GenerateSecretKey()
	if err != nil {
		return nil, err
	}

	id, err := secretKey.ID()
	if err != nil {
		return nil, err
	}

	key := &models.Key{
		ID:              id,
		Environment:     s.Environment(),
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}

	if err := s.Key().Create(key); err != nil {
		return nil, err
	}

	formattedKey, err := secretKey.Format()
	if err != nil {
		return nil, err
	}

	return &dto.RawKey{
		ID:              id,
		Environment:     s.Environment(),
		SecretKey:       formattedKey,
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}, nil

}

func (s *keyService) GetKey(id string) (*models.Key, error) {
	return s.Key().Read(id)
}

func (s *keyService) DeleteKey(id string) error {
	return s.Key().Delete(id)
}

func (s *keyService) ListKeys(opts *dto.KeyListOptions) ([]*models.Key, error) {
	return s.Key().List(opts)
}

func (s *keyService) ValidateKey(secretKey string) (*models.Key, error) {
	hashedKey := secret.Hash(secretKey)

	storedKey, err := s.GetKey(hashedKey)
	if err != nil {
		return nil, err
	}

	if subtle.ConstantTimeCompare([]byte(storedKey.ID), []byte(hashedKey)) != 1 {
		return nil, &errors.InvalidApiKey
	}
	return storedKey, nil
}
