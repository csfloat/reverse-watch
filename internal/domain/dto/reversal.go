package dto

import (
	"reverse-watch/internal/domain/models/types"
)

type ReversalListOptions struct {
	SteamID         *types.SteamID
	MarketplaceSlug *string
	Cursor          *Cursor
	Limit           *uint
}
