package repository

import (
	"reverse-watch/domain/dto"
	"reverse-watch/domain/models"
)

type ReversalRepository interface {
	Create(reversal *models.Reversal) error
	BulkCreate(reversals []*models.Reversal) error
	Read(id models.Snowflake) (*models.Reversal, error)
	Update(id models.Snowflake, updates *dto.ReversalUpdates) error
	Delete(id models.Snowflake) error
	DeleteAllUserReports(steamId models.SteamID) error
	List(opts *dto.ReversalListOptions) ([]*models.Reversal, error)
}
