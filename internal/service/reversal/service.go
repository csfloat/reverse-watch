package reversal

import (
	"fmt"

	"reverse-watch/internal/domain/dto"
	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/repository"
	"reverse-watch/internal/domain/service"
)

type reversalService struct {
	repository.PrivateRepository
	repository.PublicRepository
}

var _ service.ReversalService = (*reversalService)(nil)

func NewReversalService(privateRepo repository.PrivateRepository, publicRepo repository.PublicRepository) service.ReversalService {
	return &reversalService{
		PrivateRepository: privateRepo,
		PublicRepository:  publicRepo,
	}
}

func (s *reversalService) CreateReversals(reversals ...*models.Reversal) error {
	return s.Reversal().Create(reversals...)
}

func (s *reversalService) GetReversal(id models.Snowflake) (*models.Reversal, error) {
	return s.Reversal().Read(id)
}

func (s *reversalService) UpdateReversal(id models.Snowflake, opts *dto.ReversalUpdateOptions) error {
	if err := opts.Validate(); err != nil {
		return fmt.Errorf("invalid reversal update options: %w", err)
	}
	return s.Reversal().Update(id, opts)
}

func (s *reversalService) DeleteReversal(id models.Snowflake) error {
	return s.Reversal().Delete(id)
}

func (s *reversalService) ListReversals(opts *dto.ReversalListOptions) ([]*models.Reversal, error) {
	return s.Reversal().List(opts)
}
