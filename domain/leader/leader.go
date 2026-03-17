package leader

import "context"

type Elector interface {
	Run(ctx context.Context, onLeader func())
}
