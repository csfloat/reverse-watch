package leader

import (
	"context"
	"time"
)

type Elector interface {
	Run(ctx context.Context, lockKey uint32, period time.Duration, onWork func(ctx context.Context))
}
