package key

import (
	"crypto/subtle"
	"encoding/base64"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
	"reverse-watch/internal/errors"
	"reverse-watch/pkg/crypto"
)

type keyService struct {
	repository.PrivateRepository
}

var _ service.KeyService = (*keyService)(nil)

func NewKeyService(repo repository.PrivateRepository) service.KeyService {
	return &keyService{
		PrivateRepository: repo,
	}
}

func (s *keyService) CreateKey(marketplaceSlug string, permissions models.Permissions) (*models.RawKey, error) {
	snowflake, err := models.GenSnowflake()
	if err != nil {
		return nil, err
	}

	secret, err := crypto.GenerateSecret()
	if err != nil {
		return nil, err
	}
	salt, err := crypto.GenerateSalt()
	if err != nil {
		return nil, err
	}

	encodedSecret := base64.RawURLEncoding.EncodeToString(secret)
	encodedSalt := base64.RawURLEncoding.EncodeToString(salt)

	key := &models.Key{
		Model: models.Model{
			ID: snowflake,
		},
		KeyHash:         crypto.HashSecret(encodedSecret, encodedSalt),
		Salt:            encodedSalt,
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}

	if err := s.Key().Create(key); err != nil {
		return nil, err
	}

	return &models.RawKey{
		ID:              snowflake,
		SecretKey:       crypto.FormatAPIKey(uint64(snowflake), encodedSecret),
		MarketplaceSlug: marketplaceSlug,
		Permissions:     permissions,
	}, nil
}

func (s *keyService) GetKey(id models.Snowflake) (*models.Key, error) {
	return s.Key().Read(id)
}

func (s *keyService) UpdateKey(id models.Snowflake, opts *service.UpdateKeyOptions) error {
	return s.Key().Update(id, opts)
}

func (s *keyService) DeleteKey(id models.Snowflake) error {
	return s.Key().Delete(id)
}

func (s *keyService) ListKeys(opts *service.ListKeyOptions) ([]*models.Key, error) {
	return s.Key().List(opts)
}

func (s *keyService) ValidateKey(secretKey string) (*models.Key, error) {
	id, secret, err := crypto.ParseSecretKey(secretKey)
	if err != nil {
		return nil, err
	}

	storedKey, err := s.GetKey(models.Snowflake(id))
	if err != nil {
		return nil, err
	}

	hash := crypto.HashSecret(secret, storedKey.Salt)
	if subtle.ConstantTimeCompare([]byte(storedKey.KeyHash), []byte(hash)) != 1 {
		return nil, &errors.InvalidApiKey
	}
	return storedKey, nil
}
