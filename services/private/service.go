package private

import (
	"strings"

	"reverse-watch/errors"
	"reverse-watch/types"

	"gorm.io/gorm"
)

type Service struct {
	conn *gorm.DB
}

func NewService(conn *gorm.DB) *Service {
	return &Service{
		conn: conn,
	}
}

func GetKeyParts(secretKey string) (id string, secret string, err error) {
	parts := strings.Split(secretKey, ".")
	if len(parts) != 2 {
		return "", "", &errors.InvalidApiKey
	}

	id = strings.Replace(parts[0], "sk_live_", "", 1)
	secret = parts[1]
	return id, secret, nil
}

func (s *Service) GetKeyFromSecretKey(secretKey string) (*types.Key, error) {
	id, _, err := GetKeyParts(secretKey)
	if err != nil {
		return nil, err
	}
	return s.GetKeyFromID(id)
}

func NewKey(marketplaceSlug string, scope types.Scope) (*types.Key, error) {
	rawKey, err := NewRawKey(marketplaceSlug, scope)
	if err != nil {
		return nil, err
	}
	return rawKey.ToKey(), nil
}
