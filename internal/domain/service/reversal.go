package service

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type ReversalService interface {
	CreateReversals(reversals ...*models.Reversal) error
	GetReversal(id models.Snowflake) (*models.Reversal, error)
	UpdateReversal(id models.Snowflake, fields map[string]interface{}) error
	DeleteReversal(id models.Snowflake) error
	ListReversals(opts *dto.ReversalListOptions) ([]*models.Reversal, error)
}
