package factory

import (
	"hash/fnv"

	"reverse-watch/domain/repository"
	"reverse-watch/domain/secret"
	"reverse-watch/repository/private"
	"reverse-watch/repository/public"

	"gorm.io/gorm"
)

type privateTransaction struct {
	tx          *gorm.DB
	key         repository.KeyRepository
	marketplace repository.MarketplaceRepository
	adminAudit  repository.AdminAuditRepository
}

func newPrivateTransaction(tx *gorm.DB, keygen secret.KeyGenerator) *privateTransaction {
	return &privateTransaction{
		tx:          tx,
		key:         private.NewKeyRepository(tx, keygen),
		marketplace: private.NewMarketplaceRepository(tx),
		adminAudit:  private.NewAdminAuditRepository(tx),
	}
}

func (t *privateTransaction) Commit() error {
	return t.tx.Commit().Error
}

func (t *privateTransaction) Rollback() error {
	return t.tx.Rollback().Error
}

func (t *privateTransaction) Key() repository.KeyRepository {
	return t.key
}

func (t *privateTransaction) Marketplace() repository.MarketplaceRepository {
	return t.marketplace
}

func (t *privateTransaction) AdminAudit() repository.AdminAuditRepository {
	return t.adminAudit
}

type publicTransaction struct {
	tx       *gorm.DB
	reversal repository.ReversalRepository
}

func newPublicTransaction(tx *gorm.DB) *publicTransaction {
	return &publicTransaction{
		tx:       tx,
		reversal: public.NewReversalRepository(tx),
	}
}

func (t *publicTransaction) Commit() error {
	return t.tx.Commit().Error
}

func (t *publicTransaction) Rollback() error {
	return t.tx.Rollback().Error
}

func (t *publicTransaction) Reversal() repository.ReversalRepository {
	return t.reversal
}

func (t *publicTransaction) TryAdvisoryXactLock(id, salt string) (bool, error) {
	h := fnv.New32a()
	h.Write([]byte(salt))
	h.Write([]byte(id))
	lockKey := h.Sum32()

	var hasLock bool
	if err := t.tx.Raw("SELECT pg_try_advisory_xact_lock(?)", lockKey).Scan(&hasLock).Error; err != nil {
		return false, err
	}

	if !hasLock {
		return false, nil
	}
	return true, nil
}
