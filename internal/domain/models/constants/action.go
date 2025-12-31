package constants

type TargetAction uint

const (
	TargetActionAddMarketplace    TargetAction = 0
	TargetActionUpdateMarketplace TargetAction = 1
	TargetActionRemoveMarketplace TargetAction = 2
	TargetActionAddKey            TargetAction = 3
	TargetActionRemoveKey         TargetAction = 4
	TargetActionUpdateReversal    TargetAction = 5
	TargetActionRemoveReversal    TargetAction = 6
	TargetActionDeleteUserData    TargetAction = 7
)
