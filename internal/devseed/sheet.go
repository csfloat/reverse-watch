// Package devseed loads dev-only fixture data into the local Postgres
// instance. It is intentionally not wired into the main binary — call it
// from cmd/seed (or a test) when you need realistic data locally.
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

// insertChunkSize keeps each bulk insert under Postgres's 65,535
// parameter-per-statement cap. At ~11 columns per Reversal, 1,000 rows
// uses ~11k parameters.
const insertChunkSize = 1000

// InsertReversals bulk-inserts reversals with ON CONFLICT (id) DO NOTHING,
// so the seed is idempotent. Returns the number of rows actually inserted.
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
