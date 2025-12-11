package private

import (
	"crypto/subtle"

	"reverse-watch/errors"
	"reverse-watch/types"
)

func (s *Service) Validate(secretKey string) (*types.Key, error) {
	id, secret, err := GetKeyParts(secretKey)
	if err != nil {
		return nil, err
	}

	storedKey, err := s.GetKeyFromID(id)
	if err != nil {
		return nil, err
	}

	hash := HashSecret(secret, storedKey.Salt)
	if subtle.ConstantTimeCompare([]byte(storedKey.KeyHash), []byte(hash)) != 1 {
		return nil, &errors.InvalidApiKey
	}

	return storedKey, nil
}
