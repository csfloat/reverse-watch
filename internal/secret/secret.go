package secret

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"reverse-watch/internal/domain/models"
	"reverse-watch/internal/domain/secret"
)

var keyPrefixes = map[models.Environment]string{
	models.EnvironmentDevelopment: "reversewatch_test_",
	models.EnvironmentProduction:  "reversewatch_live_",
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
	env    models.Environment
}

var _ secret.SecretKey = (*secretKey)(nil)

func newSecretKey(env models.Environment) (secret.SecretKey, error) {
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
	return secret.Hash(key), nil
}

type keyGenerator struct {
	env models.Environment
}

var _ secret.KeyGenerator = (*keyGenerator)(nil)

func NewKeyGenerator(env models.Environment) secret.KeyGenerator {
	return &keyGenerator{
		env: env,
	}
}

func (g *keyGenerator) GenerateSecretKey() (secret.SecretKey, error) {
	return newSecretKey(g.env)
}

func (g *keyGenerator) Environment() models.Environment {
	return g.env
}
