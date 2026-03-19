package leader

import (
	"context"
	"database/sql"
	"sync"
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

func (e *elector) tryAdvisoryLock(ctx context.Context, lockKey uint32) (*sql.Conn, bool) {
	db, err := e.factory.PublicDB().DB()
	if err != nil {
		return nil, false
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, false
	}

	var hasLock bool
	err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockKey).Scan(&hasLock)
	if err != nil {
		conn.Close()
		return nil, false
	}

	if !hasLock {
		conn.Close()
		return nil, false
	}
	return conn, true
}

func (e *elector) runWithLock(ctx context.Context, conn *sql.Conn, onWork func(ctx context.Context)) {
	workCtx, workCancel := context.WithCancel(ctx)

	var wg sync.WaitGroup
	wg.Add(1)

	// Monitor connection health in background
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-workCtx.Done():
				return
			case <-ticker.C:
				if err := conn.PingContext(workCtx); err != nil {
					e.log.Warnf("failed to ping connection: %v", err)
					workCancel()
					return
				}
			}
		}
	}()

	onWork(workCtx)
	workCancel()
	wg.Wait()
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

		connCtx, connCancel := context.WithCancel(ctx)
		conn, acquired := e.tryAdvisoryLock(connCtx, lockKey)
		if !acquired {
			connCancel()
			continue
		}

		e.runWithLock(connCtx, conn, onWork)
		connCancel()
		conn.Close()
	}
}
