package secret

import (
	"reverse-watch/internal/domain/models"
)

type SecretKey interface {
	Format() (string, error)
	ID() (string, error)
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
	Environment() models.Environment
}
