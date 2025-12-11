package private

import (
	"strconv"
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

func GetKeyParts(secretKey string) (id types.Snowflake, secret string, err error) {
	parts := strings.Split(secretKey, ".")
	if len(parts) != 2 {
		return 0, "", &errors.InvalidApiKey
	}

	idStr := strings.Replace(parts[0], "sk_live_", "", 1)
	n, _ := strconv.ParseUint(idStr, 10, 64)

	return types.Snowflake(n), parts[1], nil
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
