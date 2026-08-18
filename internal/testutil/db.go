package testutil

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/domain/secret"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/peterldowns/pgtestdb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newTestDB(t)

	mods := []interface{}{
		(*models.Marketplace)(nil),
		(*models.Key)(nil),
		(*models.AdminAudit)(nil),
		(*models.Reversal)(nil),
		(*models.SearchCount)(nil),
	}

	for _, model := range mods {
		if err := db.AutoMigrate(model); err != nil {
			t.Fatalf("failed to migrate model %T: %v", model, err)
		}
	}
	return db
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	models.InitSnowflakeGenerator(0 /* workerID */, 0 /* processID */)

	cfg := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "postgres",
		Host:       "localhost",
		Port:       "5432",
		Options:    "sslmode=disable",
	}

	sqlDB := pgtestdb.New(t, cfg, pgtestdb.NoopMigrator{})
	dialector := postgres.New(postgres.Config{
		Conn: sqlDB,
	})

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	return db
}

func Insert[T any](t *testing.T, db *gorm.DB, values ...T) {
	t.Helper()

	for _, value := range values {
		if err := db.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func MustRawJsonb(value interface{}) *models.RawJsonb {
	bytes, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return &models.RawJsonb{
		Raw: bytes,
	}
}

// SetupMarketplaceWithKey sets up a test marketplace and a key with the given permissions.
func SetupMarketplaceWithKey(t *testing.T, db *gorm.DB, slug string, keygen secret.KeyGenerator, permissions models.Permissions) (*models.Marketplace, *models.Key, string) {
	testMarketplace := &models.Marketplace{
		Slug:     slug,
		Name:     "Test Marketplace",
		IsActive: true,
	}
	Insert(t, db, testMarketplace)

	secretKey, err := keygen.GenerateSecretKey()
	if err != nil {
		t.Fatalf("GenerateSecretKey(): %v", err)
	}

	id, err := secretKey.ID()
	if err != nil {
		t.Fatalf("ID(): %v", err)
	}

	authKey := &models.Key{
		ID:              id,
		MarketplaceSlug: testMarketplace.Slug,
		Environment:     keygen.Environment(),
		Permissions:     permissions,
	}
	Insert(t, db, authKey)

	formattedKey, err := secretKey.Format()
	if err != nil {
		t.Fatalf("Format(): %v", err)
	}

	return testMarketplace, authKey, formattedKey
}

// SetupAuthenticatedRequest sets up an authenticated request with the given method, path, body, and token.
func SetupAuthenticatedRequest(method string, path string, body []byte, token string) *http.Request {
	r := httptest.NewRequest(method, path, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	return r
}
