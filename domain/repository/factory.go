package repository

import "io"

type PrivateTransaction interface {
	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository

	Commit() error
	Rollback()
}

type PublicTransaction interface {
	Reversal() ReversalRepository

	Commit() error
	Rollback()
}

type Factory interface {
	io.Closer

	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
	Reversal() ReversalRepository

	NewPrivateTransaction() PrivateTransaction
	RunInTransactionPrivate(fn func(PrivateTransaction) error) error

	NewPublicTransaction() PublicTransaction
	RunInTransactionPublic(fn func(PublicTransaction) error) error
}
