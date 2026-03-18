package leader

import (
	"context"
	"time"

	"reverse-watch/config"
	"reverse-watch/domain/leader"
	"reverse-watch/domain/repository"

	"go.uber.org/zap"
)

type elector struct {
	log     *zap.SugaredLogger
	cfg     *config.Config
	factory repository.Factory
}

var _ leader.Elector = (*elector)(nil)

func New(factory repository.Factory, cfg *config.Config, log *zap.SugaredLogger) leader.Elector {
	return &elector{
		log:     log,
		cfg:     cfg,
		factory: factory,
	}
}

// Run will run the elector in a loop, acquiring the leader lock and calling the onLeader function when the leader is acquired.
// If the leader lock is not acquired, it will wait for the next minute and try again.
// If the context is done, it will return.
func (e *elector) Run(ctx context.Context, lockKey []byte, onLeader func()) {
	for {
		now := time.Now()
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextMinute)):
		}

		conn, err := e.factory.PrivateDB().DB()
		if err != nil {
			e.log.Errorf("failed to get private database connection: %v", err)
			continue
		}

		var hasLock bool
		err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(?)", lockKey).Scan(&hasLock)
		if err != nil {
			e.log.Errorf("failed to check if leader lock is acquired: %v", err)
			continue
		}

		if hasLock {
			e.log.Info("leader lock acquired")
			onLeader()
			return
		}
	}
}
