package public

import "gorm.io/gorm"

type Service struct {
	conn *gorm.DB
}

func NewService(conn *gorm.DB) *Service {
	return &Service{
		conn: conn,
	}
}
