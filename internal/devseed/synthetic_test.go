package devseed

import (
	"testing"
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/internal/testutil"
	"reverse-watch/repository/public"
)

// Re-running the seed produces new snowflake IDs but the same
// (steam_id, marketplace_slug) pairs, which collide with the partial unique
// index. The insert must skip those rows via ON CONFLICT DO NOTHING rather
// than raising a unique-constraint error.
func TestInsertReversals_RerunIsSafe(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := public.CreateIndexes(db); err != nil {
		t.Fatalf("CreateIndexes: %v", err)
	}

	// Both runs use past timestamps (BeforeCreate rejects future
	// reversed_at), but on different calendar days so snowflake IDs differ.
	now := time.Now().UTC().AddDate(0, 0, -10)
	if _, err := InsertReversals(db, GenerateSynthetic(now)); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	n2, err := InsertReversals(db, GenerateSynthetic(now.AddDate(0, 0, 5)))
	if err != nil {
		t.Fatalf("rerun errored, expected ON CONFLICT DO NOTHING to skip: %v", err)
	}
	if n2 != 0 {
		t.Errorf("rerun inserted %d rows, want 0", n2)
	}
}

// ResetSynthetic must remove every synthetic row, including soft-deleted
// ones, so a reseed can insert a fresh dataset ending at the new date. Rows
// outside the synthetic Steam ID range must be left alone.
func TestResetSynthetic_RefreshesData(t *testing.T) {
	db := testutil.NewTestDB(t)
	if err := public.CreateIndexes(db); err != nil {
		t.Fatalf("CreateIndexes: %v", err)
	}

	nonSynthetic := &models.Reversal{
		Model:           models.Model{ID: 1},
		SteamID:         models.SteamID(76561197960287930),
		MarketplaceSlug: "csfloat",
	}
	testutil.Insert(t, db, nonSynthetic)

	old := time.Now().UTC().AddDate(0, 0, -10)
	first := GenerateSynthetic(old)
	if _, err := InsertReversals(db, first); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if err := db.Delete(&models.Reversal{}, first[0].ID).Error; err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	deleted, err := ResetSynthetic(db)
	if err != nil {
		t.Fatalf("ResetSynthetic: %v", err)
	}
	if deleted != int64(len(first)) {
		t.Errorf("ResetSynthetic deleted %d rows, want %d", deleted, len(first))
	}

	fresh := GenerateSynthetic(time.Now().UTC())
	inserted, err := InsertReversals(db, fresh)
	if err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if inserted != int64(len(fresh)) {
		t.Errorf("reseed inserted %d rows, want %d", inserted, len(fresh))
	}

	var realCount int64
	if err := db.Model(&models.Reversal{}).Where("id = ?", nonSynthetic.ID).Count(&realCount).Error; err != nil {
		t.Fatalf("count real row: %v", err)
	}
	if realCount != 1 {
		t.Errorf("non-synthetic row was deleted by ResetSynthetic")
	}
}

// Synthetic rows must use unallocated Steam IDs and must never share a slug
// or reporter ID with a real ingestor, otherwise they would corrupt the
// CSFloat ingestor's resume cursor.
func TestGenerateSynthetic_IsolatedFromRealData(t *testing.T) {
	now := time.Now().UTC()
	rows := GenerateSynthetic(now)
	for _, r := range rows {
		ids := []models.SteamID{r.SteamID}
		if r.RelatedSteamID != nil {
			ids = append(ids, *r.RelatedSteamID)
		}
		for _, id := range ids {
			if !id.IsValid() {
				t.Fatalf("steam id %d is not valid", id)
			}
			if uint64(id) < SyntheticSteamIDMin || uint64(id) > SyntheticSteamIDMax {
				t.Fatalf("steam id %d is outside the synthetic range", id)
			}
		}
		if r.MarketplaceSlug == "csfloat" {
			t.Fatalf("row %d uses the real csfloat slug", r.ID)
		}
		if r.ReporterInternalID != nil {
			t.Fatalf("row %d has a reporter_internal_id", r.ID)
		}
		if r.ReversedAt > uint64(now.UnixMilli()) || r.CreatedAt > uint64(now.UnixMilli()) {
			t.Fatalf("row %d has a timestamp after now", r.ID)
		}
	}
}

// Seeding early in the day must spread today's rows across the elapsed part
// of the day instead of clamping them all to the same minute.
func TestGenerateSynthetic_TodayNotClamped(t *testing.T) {
	now := time.Date(2026, 6, 1, 6, 0, 0, 0, time.UTC)
	dayStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	seen := map[int64]bool{}
	todayRows := 0
	for _, r := range GenerateSynthetic(now) {
		reversedAt := time.UnixMilli(int64(r.ReversedAt)).UTC()
		if reversedAt.Before(dayStart) {
			continue
		}
		todayRows++
		seen[reversedAt.Truncate(time.Minute).Unix()] = true
	}
	if todayRows < 2 {
		t.Skipf("only %d rows generated for today", todayRows)
	}
	if len(seen) < 2 {
		t.Errorf("all %d of today's rows share one minute", todayRows)
	}
}

// Generating data with a clock that predates models.Epoch must not
// underflow the unsigned snowflake timestamp subtraction. Every decoded
// timestamp must be >= Epoch.
func TestGenerateSynthetic_NoEpochUnderflow(t *testing.T) {
	// A clock one day before Epoch makes every generated createdAt precede
	// Epoch, so the guard must clamp each snowflake timestamp to exactly
	// Epoch. Without the guard the unsigned subtraction wraps to a huge
	// value, decoding to a timestamp far beyond Epoch.
	beforeEpoch := time.UnixMilli(int64(models.Epoch)).UTC().AddDate(0, 0, -1)
	rows := GenerateSynthetic(beforeEpoch)
	if len(rows) == 0 {
		t.Fatal("expected synthetic rows")
	}
	for _, r := range rows {
		ts := models.ParseSnowflake(r.ID).Timestamp
		if ts != models.Epoch {
			t.Fatalf("snowflake timestamp %d != Epoch %d (underflow not guarded)", ts, models.Epoch)
		}
	}
}
