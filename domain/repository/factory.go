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
	SearchCount() SearchCountRepository
}

type Factory interface {
	io.Closer

	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
	Reversal() ReversalRepository
	SearchCount() SearchCountRepository

	NewPrivateTransaction() PrivateTransaction
	RunInTransactionPrivate(fn func(PrivateTransaction) error) error
	PrivateDB() *gorm.DB

	NewPublicTransaction() PublicTransaction
	RunInTransactionPublic(fn func(PublicTransaction) error) error
	PublicDB() *gorm.DB
}
