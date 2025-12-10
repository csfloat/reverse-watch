package private

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"reverse-watch/types"

	"github.com/google/uuid"
)

type RawKey struct {
	ID              string `json:"id"`
	secret          string
	SecretKey       string      `json:"secret_key"`
	MarketplaceSlug string      `json:"marketplace_slug"`
	Scope           types.Scope `json:"scope"`
}

func NewRawKey(marketplaceSlug string, scope types.Scope) (*RawKey, error) {
	return generateRawKey(marketplaceSlug, scope)
}

func (r *RawKey) ToKey() *types.Key {
	return &types.Key{
		ID:              r.ID,
		KeyHash:         HashSecret(r.secret),
		MarketplaceSlug: r.MarketplaceSlug,
		Scope:           r.Scope,
	}
}

func generateRawKey(marketplaceSlug string, scope types.Scope) (*RawKey, error) {
	rawKey := &RawKey{
		ID:              uuid.New().String(),
		MarketplaceSlug: marketplaceSlug,
		Scope:           scope,
	}

	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}

	rawKey.secret = base64.RawURLEncoding.EncodeToString(bytes)
	rawKey.SecretKey = fmt.Sprintf("sk_live_%s.%s", rawKey.ID, rawKey.secret)

	return rawKey, nil
}

func HashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}
