package secret

import "reverse-watch/internal/config"

type SecretKey interface {
	Format() (string, error)
	ID() (string, error)
}

type KeyGenerator interface {
	GenerateSecretKey() (SecretKey, error)
	Environment() config.Environment
}
