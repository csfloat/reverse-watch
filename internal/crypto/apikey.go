package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"reverse-watch/internal/domain/models"
)

func HashSecret(secret, salt string) string {
	h := hmac.New(sha256.New, []byte(salt))
	h.Write([]byte(secret))
	return hex.EncodeToString(h.Sum(nil))
}

func ParseSecretKey(secretKey string) (id models.Snowflake, secret string, err error) {
	parts := strings.Split(secretKey, ".")
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid secret key")
	}

	idStr := strings.Replace(parts[0], "sk_live_", "", 1)
	n, _ := strconv.ParseUint(idStr, 10, 64)

	return models.Snowflake(n), parts[1], nil
}

func GenerateSecretKey() (id models.Snowflake, secret []byte, salt []byte, err error) {
	id, err = models.GenSnowflake()
	if err != nil {
		return 0, nil, nil, err
	}

	secret, err = generateRandomBytes(32)
	if err != nil {
		return 0, nil, nil, err
	}
	salt, err = generateRandomBytes(16)
	if err != nil {
		return 0, nil, nil, err
	}
	return id, secret, salt, nil
}

func generateRandomBytes(length uint) ([]byte, error) {
	if length == 0 {
		return nil, fmt.Errorf("length must greater than zero")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

func FormatAPIKey(id uint64, secret string) string {
	return fmt.Sprintf("sk_live_%d.%s", id, secret)
}
