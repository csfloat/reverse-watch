package repository

import (
	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
)

type ReversalRepository interface {
	Create(reversal *models.Reversal) error
	BulkCreate(reversals []*models.Reversal) error
	Read(id models.Snowflake) (*models.Reversal, error)
	Update(id models.Snowflake, opts *dto.ReversalUpdateOptions) error
	Delete(id models.Snowflake) error
	List(opts *dto.ReversalListOptions) ([]*models.Reversal, error)
}

type PublicRepository interface {
	Reversal() ReversalRepository
	Close() error
}
