package private

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"reverse-watch/errors"
	"reverse-watch/types"
)

type RawKey struct {
	ID              types.Snowflake `json:"id"`
	secret          string
	SecretKey       string      `json:"secret_key"`
	Salt            string      `json:"-"`
	MarketplaceSlug string      `json:"marketplace_slug"`
	Scope           types.Scope `json:"scope"`
}

func NewRawKey(marketplaceSlug string, scope types.Scope) (*RawKey, error) {
	return generateRawKey(marketplaceSlug, scope)
}

func (r *RawKey) ToKey() *types.Key {
	return &types.Key{
		Model: types.Model{
			ID: r.ID,
		},
		KeyHash:         HashSecret(r.secret, r.Salt),
		Salt:            r.Salt,
		MarketplaceSlug: r.MarketplaceSlug,
		Scope:           r.Scope,
	}
}

func generateRandomBytes(length uint) ([]byte, error) {
	if length == 0 {
		return nil, errors.New(errors.InternalServerError, "length must be greater than zero")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func generateRawKey(marketplaceSlug string, scope types.Scope) (*RawKey, error) {
	snowflake, err := types.GenSnowflake()
	if err != nil {
		return nil, err
	}

	rawKey := &RawKey{
		ID:              snowflake,
		MarketplaceSlug: marketplaceSlug,
		Scope:           scope,
	}

	secret, err := generateRandomBytes(32)
	if err != nil {
		return nil, err
	}

	salt, err := generateRandomBytes(16)
	if err != nil {
		return nil, err
	}

	rawKey.secret = base64.RawURLEncoding.EncodeToString(secret)
	rawKey.Salt = base64.RawURLEncoding.EncodeToString(salt)
	rawKey.SecretKey = fmt.Sprintf("sk_live_%s.%s", rawKey.ID, rawKey.secret)

	return rawKey, nil
}

func HashSecret(secret, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum(nil))
}
