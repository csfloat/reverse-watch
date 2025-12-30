package dto

import (
	"reverse-watch/internal/domain/models"
)

// RawKey represents an unhashed API key returned to the consumer upon creation
type RawKey struct {
	// ID is the key's hash
	ID              string             `json:"id"`
	Environment     models.Environment `json:"environment"`
	SecretKey       string             `json:"secret_key"`
	MarketplaceSlug string             `json:"marketplace_slug"`
	Permissions     models.Permissions `json:"permissions"`
}

type KeyUpdateOptions struct {
	Permissions *models.Permissions
}

type KeyListOptions struct {
	MarketplaceSlug *string
}
