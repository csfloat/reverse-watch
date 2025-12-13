package repository

import (
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/service"
)

type ReversalRepository interface {
	Create(reversal *models.Reversal) error
	Read(id models.Snowflake) (*models.Reversal, error)
	Update(id models.Snowflake, fields map[string]interface{}) error
	Delete(id models.Snowflake) error
	List(opts *service.ListReversalOptions) ([]*models.Reversal, error)
}

type PublicRepository interface {
	Reversal() ReversalRepository
}
