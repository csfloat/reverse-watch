package leader

import "context"

type Elector interface {
	Run(ctx context.Context, lockKey uint32, onLeader func())
}
