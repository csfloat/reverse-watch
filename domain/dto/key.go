package dto

import (
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
)

// RawKey represents an unhashed API key returned to the consumer upon creation
type RawKey struct {
	// ID is the key's hash
	ID              string                `json:"id"`
	Environment     constants.Environment `json:"environment"`
	SecretKey       string                `json:"secret_key"`
	MarketplaceSlug string                `json:"marketplace_slug"`
	Permissions     models.Permissions    `json:"permissions"`
}

type KeyListOptions struct {
	MarketplaceSlug *string
}
