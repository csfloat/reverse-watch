package repository

import "io"

type PrivateTransaction interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository

	Commit() error
	Rollback() error
}

type PublicTransaction interface {
	Reversal() ReversalRepository

	Commit() error
	Rollback() error
}

type Factory interface {
	io.Closer

	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
	Reversal() ReversalRepository

	RunInTransactionPrivate(fn func(PrivateTransaction) error) error
	RunInTransactionPublic(fn func(PublicTransaction) error) error
}
