package reversal

import (
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

func (s *reversalService) CreateReversal(reversal *models.Reversal) error {
	return s.Reversal().Create(reversal)
}

func (s *reversalService) BulkCreateReversals(reversals []*models.Reversal) error {
	return s.Reversal().BulkCreate(reversals)
}

func (s *reversalService) GetReversal(id models.Snowflake) (*models.Reversal, error) {
	return s.Reversal().Read(id)
}

func (s *reversalService) UpdateReversal(id models.Snowflake, fields map[string]interface{}) error {
	return s.Reversal().Update(id, fields)
}

func (s *reversalService) DeleteReversal(id models.Snowflake) error {
	return s.Reversal().Delete(id)
}

func (s *reversalService) ListReversals(opts service.ReversalListOptions) ([]*models.Reversal, error) {
	return s.Reversal().List(opts)
}
