package service

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type ReversalService interface {
	CreateReversal(reversal *models.Reversal) error
	BulkCreateReversals(reversals []*models.Reversal) error
	GetReversal(id models.Snowflake) (*models.Reversal, error)
	UpdateReversal(id models.Snowflake, opts *dto.ReversalUpdates) error
	DeleteReversal(id models.Snowflake) error
	ListReversals(opts *dto.ReversalListOptions) ([]*models.Reversal, error)
}
