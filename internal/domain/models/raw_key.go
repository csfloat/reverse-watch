package models

// RawKey represents an unhashed API key returned to the consumer upon creation
type RawKey struct {
	// ID is the key's hash
	ID              string      `json:"id"`
	Environment     string      `json:"environment"`
	SecretKey       string      `json:"secret_key"`
	MarketplaceSlug string      `json:"marketplace_slug"`
	Permissions     Permissions `json:"permissions"`
}
