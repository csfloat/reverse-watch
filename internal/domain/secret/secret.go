package secret

import (
	"reverse-watch/internal/domain/models/constants"
)

type SecretKey interface {
	Format() (string, error)
	ID() (string, error)
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
	Environment() constants.Environment
}
