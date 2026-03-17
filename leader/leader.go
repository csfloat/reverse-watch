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
func (e *elector) Run(ctx context.Context, onLeader func()) {
	for {
		now := time.Now()
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)

		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(nextMinute)):
		}

		tx := e.factory.NewPublicTransaction()
		acquired, err := tx.TryAdvisoryXactLock(e.cfg.Elector.ID, e.cfg.Elector.Salt)
		if err != nil {
			e.log.Warnf("failed to acquire leader lock: %v", err)
			tx.Rollback()
			continue
		}

		if !acquired {
			e.log.Warn("failed to acquire leader lock")
			tx.Rollback()
			continue
		}

		e.log.Info("leader lock acquired")
		onLeader()

		tx.Rollback()
		return
	}
}
