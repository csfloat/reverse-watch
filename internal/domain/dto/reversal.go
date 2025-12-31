package dto

import (
	"reverse-watch/internal/domain/models"
)

type ReversalListOptions struct {
	SteamID         *models.SteamID
	MarketplaceSlug *string
	Cursor          *Cursor
	Limit           *uint
}
