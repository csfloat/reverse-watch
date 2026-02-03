package repository

import "io"

type Factory interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
	Reversal() ReversalRepository
	io.Closer
}
