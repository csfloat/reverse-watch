package dto

import (
	"reverse-watch/internal/domain/models/types"
)

// RawKey represents an unhashed API key returned to the consumer upon creation
type RawKey struct {
	// ID is the key's hash
	ID              string            `json:"id"`
	Environment     types.Environment `json:"environment"`
	SecretKey       string            `json:"secret_key"`
	MarketplaceSlug string            `json:"marketplace_slug"`
	Permissions     types.Permissions `json:"permissions"`
}

type KeyUpdateOptions struct {
	Permissions *types.Permissions
}

type KeyListOptions struct {
	MarketplaceSlug *string
}
