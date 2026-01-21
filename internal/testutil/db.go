package testutil

import (
	"encoding/json"
	"testing"

	"reverse-watch/internal/domain/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPrivateTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newTestDB(t)
	privateModels := []interface{}{
		(*models.Marketplace)(nil),
		(*models.Key)(nil),
		(*models.AdminAudit)(nil),
	}

	for _, model := range privateModels {
		if err := db.AutoMigrate(model); err != nil {
			t.Fatalf("failed to migrate model %T: %v", model, err)
		}
	}
	return db
}

func NewPublicTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := newTestDB(t)
	publicModels := []interface{}{
		(*models.Reversal)(nil),
	}

	for _, model := range publicModels {
		if err := db.AutoMigrate(model); err != nil {
			t.Fatalf("failed to migrate model %T: %v", model, err)
		}
	}
	return db
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	models.InitSnowflakeGenerator(0 /* workerID */, 0 /* processID */)

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatalf("failed to enable foreign keys: %v", err)
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
