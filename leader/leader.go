package leader

import (
	"context"
	"database/sql"
	"time"

	"reverse-watch/domain/leader"
	"reverse-watch/domain/repository"

	"go.uber.org/zap"
)

type elector struct {
	log     *zap.SugaredLogger
	factory repository.Factory
}

var _ leader.Elector = (*elector)(nil)

func New(factory repository.Factory, log *zap.SugaredLogger) leader.Elector {
	return &elector{
		log:     log,
		factory: factory,
	}
}

func (e *elector) tryAdvisoryLock(ctx context.Context, lockKey uint32) (*sql.Tx, bool) {
	db, err := e.factory.PublicDB().DB()
	if err != nil {
		return nil, false
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false
	}

	var hasLock bool
	err = tx.QueryRowContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", lockKey).Scan(&hasLock)
	if err != nil {
		tx.Rollback()
		return nil, false
	}

	if !hasLock {
		tx.Rollback()
		return nil, false
	}
	return tx, true
}

// Run will run the elector in a loop, acquiring the leader lock and calling the onWork function when the leader is acquired.
// If the leader lock is not acquired, it will wait for the next period and try again.
// If the context is done, it will return.
func (e *elector) Run(ctx context.Context, lockKey uint32, period time.Duration, onWork func(ctx context.Context)) {
	for {
		nextBoundary := time.Now().Truncate(period).Add(period)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextBoundary)):
		}

		txCtx, txCancel := context.WithCancel(ctx)
		tx, acquired := e.tryAdvisoryLock(txCtx, lockKey)
		if !acquired {
			txCancel()
			continue
		}

		onWork(txCtx)
		tx.Rollback()
		txCancel()
	}
}
