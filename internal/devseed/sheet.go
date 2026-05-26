// Package devseed loads dev-only fixture data into the local Postgres
// instance. It is intentionally NOT wired into the main binary — call it
// from cmd/seed (or a test) when you need realistic data locally.
//
// The fixture CSV at fixtures/reversals_seed.csv is a 98-row export of
// the "Reverse Watch - Studio Results 2026-05-22 11:13" Google Sheet.
// Two rows from the original 100-row export had steam_id precision loss
// (see docs/dashboard/HANDOFF.md §10) and were dropped before commit.
package devseed

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"reverse-watch/domain/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// expectedHeader is the exact column ordering the CSV must use. We match
// on order, not name, but we verify the names so a re-export with
// shuffled columns fails loudly instead of silently mis-mapping fields.
var expectedHeader = []string{
	"id",
	"created_at",
	"updated_at",
	"steam_id",
	"marketplace_slug",
	"source",
	"related_steam_id",
	"reversed_at",
	"reporter_internal_id",
	"expunged_at",
}

// LoadFromCSV reads the dev-seed CSV at path and returns reversals ready
// to insert. The returned models have their IDs and timestamps populated
// from the CSV — the model hooks will not override them.
func LoadFromCSV(path string) ([]*models.Reversal, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	// Tolerate trailing empty fields (e.g. when expunged_at is blank).
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if err := validateHeader(header); err != nil {
		return nil, err
	}

	var out []*models.Reversal
	line := 1
	for {
		line++
		rec, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read line %d: %w", line, err)
		}
		r, err := parseRow(rec, line)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func validateHeader(got []string) error {
	if len(got) != len(expectedHeader) {
		return fmt.Errorf("header: got %d columns, want %d", len(got), len(expectedHeader))
	}
	for i, name := range expectedHeader {
		if got[i] != name {
			return fmt.Errorf("header column %d: got %q, want %q", i, got[i], name)
		}
	}
	return nil
}

func parseRow(rec []string, line int) (*models.Reversal, error) {
	// Pad short records (trailing empty columns get trimmed by csv.Reader).
	for len(rec) < len(expectedHeader) {
		rec = append(rec, "")
	}

	id, err := strconv.ParseUint(rec[0], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: id: %w", line, err)
	}
	createdAt, err := strconv.ParseUint(rec[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: created_at: %w", line, err)
	}
	updatedAt, err := strconv.ParseUint(rec[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: updated_at: %w", line, err)
	}
	steamID, err := models.ToSteamID(rec[3])
	if err != nil {
		return nil, fmt.Errorf("line %d: steam_id: %w", line, err)
	}
	marketplace := rec[4]
	if marketplace == "" {
		return nil, fmt.Errorf("line %d: marketplace_slug is required", line)
	}
	srcRaw, err := strconv.ParseUint(rec[5], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: source: %w", line, err)
	}
	src := models.Source(srcRaw)

	var related *models.SteamID
	if rec[6] != "" {
		related, err = models.ToSteamID(rec[6])
		if err != nil {
			return nil, fmt.Errorf("line %d: related_steam_id: %w", line, err)
		}
	}

	reversedAt, err := strconv.ParseUint(rec[7], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("line %d: reversed_at: %w", line, err)
	}

	var reporterInternalID *uint
	if rec[8] != "" {
		rip, err := strconv.ParseUint(rec[8], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: reporter_internal_id: %w", line, err)
		}
		v := uint(rip)
		reporterInternalID = &v
	}

	var expungedAt *uint64
	if rec[9] != "" {
		ea, err := strconv.ParseUint(rec[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: expunged_at: %w", line, err)
		}
		expungedAt = &ea
	}

	return &models.Reversal{
		Model: models.Model{
			ID:        models.Snowflake(id),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		SteamID:            *steamID,
		MarketplaceSlug:    marketplace,
		Source:             &src,
		RelatedSteamID:     related,
		ReversedAt:         reversedAt,
		ReporterInternalID: reporterInternalID,
		ExpungedAt:         expungedAt,
	}, nil
}

// insertChunkSize is the number of rows GORM bulk-inserts per round trip.
// Postgres caps a single statement at 65,535 bound parameters (uint16);
// at ~11 columns per Reversal row, 1,000 rows uses ~11k parameters — well
// under the limit and big enough to keep network round-trips negligible
// for the ~10k-row synthetic seed.
const insertChunkSize = 1000

// InsertReversals inserts the given reversals into the public DB using
// ON CONFLICT (id) DO NOTHING, so the seed is idempotent — running it
// twice in a row leaves the DB in the same state as running it once.
// Inserts are chunked to stay under Postgres's parameter-per-statement
// limit (see insertChunkSize). Returns the number of rows the DB
// actually inserted; rows with matching IDs are silently skipped.
func InsertReversals(db *gorm.DB, reversals []*models.Reversal) (int64, error) {
	if len(reversals) == 0 {
		return 0, nil
	}
	var inserted int64
	for i := 0; i < len(reversals); i += insertChunkSize {
		end := i + insertChunkSize
		if end > len(reversals) {
			end = len(reversals)
		}
		res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(reversals[i:end])
		if res.Error != nil {
			return inserted, res.Error
		}
		inserted += res.RowsAffected
	}
	return inserted, nil
}
