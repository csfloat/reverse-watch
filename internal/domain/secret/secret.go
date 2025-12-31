package secret

import (
	"crypto/sha256"
	"encoding/hex"

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

func Hash(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
