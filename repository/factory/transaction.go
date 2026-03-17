package factory

import (
	"context"
	"database/sql"
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
	lockKey := advisoryLockKey(id, salt)
	var hasLock bool
	if err := t.tx.Raw("SELECT pg_try_advisory_lock(?)", lockKey).Scan(&hasLock).Error; err != nil {
		return false, err
	}

	if !hasLock {
		return false, nil
	}
	return true, nil
}

type publicAdvisoryLockSession struct {
	conn *sql.Conn
}

func newPublicAdvisoryLockSession(conn *sql.Conn) *publicAdvisoryLockSession {
	return &publicAdvisoryLockSession{conn: conn}
}

func (s *publicAdvisoryLockSession) Close() error {
	return s.conn.Close()
}

func (s *publicAdvisoryLockSession) TryAdvisoryLock(id, salt string) (bool, error) {
	lockKey := advisoryLockKey(id, salt)
	var hasLock bool
	if err := s.conn.QueryRowContext(context.Background(), "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&hasLock); err != nil {
		return false, err
	}
	return hasLock, nil
}

func (s *publicAdvisoryLockSession) AdvisoryUnlock(id, salt string) error {
	lockKey := advisoryLockKey(id, salt)
	var unlocked bool
	if err := s.conn.QueryRowContext(context.Background(), "SELECT pg_advisory_unlock($1)", lockKey).Scan(&unlocked); err != nil {
		return err
	}
	return nil
}

func advisoryLockKey(id, salt string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(salt))
	h.Write([]byte(id))
	return h.Sum32()
}
