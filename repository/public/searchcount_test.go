package public

import (
	"testing"

	"reverse-watch/domain/models"
	"reverse-watch/internal/testutil"
)

func TestSearchCountRepository_Increment(t *testing.T) {
	t.Parallel()

	db := testutil.NewTestDB(t)
	repo := NewSearchCountRepository(db)

	steamID := models.SteamID(76561197960287930)

	// First lookup inserts a fresh row with count 1.
	if err := repo.Increment(steamID); err != nil {
		t.Fatalf("Increment(): %v", err)
	}

	var first models.SearchCount
	if err := db.Where("steam_id = ?", uint64(steamID)).First(&first).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}
	if first.SteamID != steamID {
		t.Errorf("SteamID = %d, want %d", first.SteamID, steamID)
	}
	if first.Count != 1 {
		t.Errorf("Count = %d, want 1", first.Count)
	}
	if first.LastSearchedAt == 0 {
		t.Errorf("LastSearchedAt = 0, want non-zero")
	}

	// Subsequent lookups upsert and increment the same row.
	if err := repo.Increment(steamID); err != nil {
		t.Fatalf("Increment(): %v", err)
	}
	if err := repo.Increment(steamID); err != nil {
		t.Fatalf("Increment(): %v", err)
	}

	var second models.SearchCount
	if err := db.Where("steam_id = ?", uint64(steamID)).First(&second).Error; err != nil {
		t.Fatalf("First(): %v", err)
	}
	if second.Count != 3 {
		t.Errorf("Count = %d, want 3", second.Count)
	}
	if second.LastSearchedAt < first.LastSearchedAt {
		t.Errorf("LastSearchedAt = %d, want >= %d", second.LastSearchedAt, first.LastSearchedAt)
	}

	// Distinct Steam IDs get their own rows.
	otherID := models.SteamID(76561197960287931)
	if err := repo.Increment(otherID); err != nil {
		t.Fatalf("Increment(): %v", err)
	}

	var rows int64
	if err := db.Model(&models.SearchCount{}).Count(&rows).Error; err != nil {
		t.Fatalf("Count(): %v", err)
	}
	if rows != 2 {
		t.Errorf("rows = %d, want 2", rows)
	}
}
