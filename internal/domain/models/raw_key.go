package models

// RawKey represents an unhashed API key returned to the consumer upon creation
type RawKey struct {
	ID              Snowflake `json:"id"`
	SecretKey       string    `json:"secret_key"` // "sk_live_{id}.{secret}"
	MarketplaceSlug string    `json:"marketplace_slug"`
	Scope           Scope     `json:"scope"`
}
