package public

import (
	"reverse-watch/types"

	"gorm.io/gorm"
)

type Service struct {
	conn *gorm.DB
}

func NewService(conn *gorm.DB) *Service {
	return &Service{
		conn: conn,
	}
}

func (s *Service) CreateReversal(reversal *types.Reversal) error {
	return s.conn.Create(reversal).Error
}
