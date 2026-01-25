package secret

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"reverse-watch/domain/models/constants"
	"reverse-watch/domain/secret"
)

var keyPrefixes = map[constants.Environment]string{
	constants.EnvironmentDevelopment: "reversewatch_test_",
	constants.EnvironmentProduction:  "reversewatch_live_",
}

func generateRandomBytes(length uint) ([]byte, error) {
	if length == 0 {
		return nil, fmt.Errorf("length must be greater than zero")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

type secretKey struct {
	secret string
	env    constants.Environment
}

var _ secret.SecretKey = (*secretKey)(nil)

func newSecretKey(env constants.Environment) (secret.SecretKey, error) {
	bytes, err := generateRandomBytes(32)
	if err != nil {
		return nil, err
	}
	return &secretKey{
		secret: base64.RawURLEncoding.EncodeToString(bytes),
		env:    env,
	}, nil
}

func (s *secretKey) Format() (string, error) {
	prefix, ok := keyPrefixes[s.env]
	if !ok {
		return "", fmt.Errorf("prefix doesn't exist for the given environment")
	}
	return fmt.Sprintf("%s%s", prefix, s.secret), nil
}

func (s *secretKey) ID() (string, error) {
	key, err := s.Format()
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:]), nil
}

type keyGenerator struct {
	env constants.Environment
}

var _ secret.KeyGenerator = (*keyGenerator)(nil)

func NewKeyGenerator(env constants.Environment) secret.KeyGenerator {
	return &keyGenerator{
		env: env,
	}
}

func (g *keyGenerator) GenerateSecretKey() (secret.SecretKey, error) {
	return newSecretKey(g.env)
}

func (g *keyGenerator) Environment() constants.Environment {
	return g.env
}
