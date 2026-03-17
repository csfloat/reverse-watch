package repository

import (
	"context"
	"io"

	"gorm.io/gorm"
)

type PrivateTransaction interface {
	gorm.TxCommitter

	Key() KeyRepository
	Marketplace() MarketplaceRepository
	AdminAudit() AdminAuditRepository
}

type AdvisoryLock interface {
	TryAdvisoryXactLock(id, salt string) (bool, error)
}

type AdvisoryLockSession interface {
	io.Closer
	TryAdvisoryLock(id, salt string) (bool, error)
	AdvisoryUnlock(id, salt string) error
}

type PublicTransaction interface {
	gorm.TxCommitter

	Reversal() ReversalRepository
	AdvisoryLock
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
	NewPublicAdvisoryLockSession(ctx context.Context) (AdvisoryLockSession, error)
}
