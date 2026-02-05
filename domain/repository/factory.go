package repository

import (
	"io"

	"gorm.io/gorm"
)

type PrivateTransaction interface {
	gorm.TxCommitter

	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
}

type PublicTransaction interface {
	gorm.TxCommitter

	Reversal() ReversalRepository
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
