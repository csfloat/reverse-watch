package devseed

import (
	"testing"
	"time"

	"reverse-watch/domain/models"
	"reverse-watch/internal/testutil"
	"reverse-watch/repository/public"
)

// TestInsertReversals_RerunIsSafe guards Finding 1: re-running the seed on a
// different calendar day produces new snowflake IDs but the same deterministic
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
	first := GenerateSynthetic(now)
	if _, err := InsertReversals(db, first); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	later := now.AddDate(0, 0, 5)
	second := GenerateSynthetic(later)
	n2, err := InsertReversals(db, second)
	if err != nil {
		t.Fatalf("rerun errored, expected ON CONFLICT DO NOTHING to skip: %v", err)
	}
	if n2 != 0 {
		t.Errorf("rerun inserted %d rows, want 0 (idempotent)", n2)
	}

	if first[0].ID == second[0].ID {
		t.Errorf("expected differing snowflake IDs across days, got %d for both", first[0].ID)
	}
}

// TestGenerateSynthetic_NoEpochUnderflow guards Finding 2: generating data
// with a clock that predates models.Epoch must not underflow the unsigned
// snowflake timestamp subtraction. Every decoded timestamp must be >= Epoch.
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
