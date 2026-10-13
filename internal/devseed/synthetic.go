// Package devseed loads dev-only fixture data into the local Postgres
// instance. It is intentionally not wired into the main binary — call it
// from cmd/seed (or a test) when you need realistic data locally.
package devseed

import (
	"math"
	"math/rand"
	"time"

	"reverse-watch/domain/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Synthetic Steam IDs live at the top of the 32-bit account space
// (account IDs >= 0xF0000000), which Valve has not allocated. They still
// pass SteamID.IsValid, and the dedicated range lets ResetSynthetic delete
// exactly the rows this package generated.
const (
	steamID64Base        uint64 = 76561197960265728
	SyntheticSteamIDMin  uint64 = steamID64Base + 0xF0000000
	SyntheticSteamIDMax  uint64 = steamID64Base + 0xFFFFFFFF
	syntheticRNGSeed     int64  = 42
	syntheticDays        int    = 180
	syntheticTargetTotal int    = 2000
)

// Marketplace slugs are prefixed with "dev-" so synthetic rows never match a
// real ingestor's slug. The CSFloat ingestor resumes from the highest
// reporter_internal_id on "csfloat" rows, so sharing that slug would corrupt
// its cursor.
var syntheticMarketplaces = []struct {
	slug   string
	weight float64
}{
	{"dev-csfloat", 0.80},
	{"dev-tradeit", 0.10},
	{"dev-skinport", 0.05},
	{"dev-swap.gg", 0.05},
}

// GenerateSynthetic returns a ~6-month dataset (~2,000 rows, at least one
// per day, gentle sinusoid with occasional spikes / quiet days) ending at
// now. Snowflake IDs are unique within the slice, so callers can pipe the
// result straight into InsertReversals.
func GenerateSynthetic(now time.Time) []*models.Reversal {
	now = now.UTC()
	rng := rand.New(rand.NewSource(syntheticRNGSeed))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	counts := make([]int, syntheticDays)
	for d := 0; d < syntheticDays; d++ {
		base := 40.0 + 10.0*math.Sin(float64(d)/30.0)
		var mult float64
		switch r := rng.Float64(); {
		case r < 0.05:
			mult = 2.5 + rng.Float64()*2.5
		case r < 0.15:
			mult = 0.2 + rng.Float64()*0.3
		default:
			mult = 0.7 + rng.Float64()*0.6
		}
		counts[d] = int(math.Max(1, base*mult+rng.NormFloat64()*5))
	}

	total := 0
	for _, c := range counts {
		total += c
	}
	if total > 0 {
		scale := float64(syntheticTargetTotal) / float64(total)
		for d := range counts {
			counts[d] = int(math.Max(1, math.Round(float64(counts[d])*scale)))
		}
	}

	rows := make([]*models.Reversal, 0, syntheticTargetTotal+200)
	var steamOffset uint64 = 1
	var seq uint16

	for d := 0; d < syntheticDays; d++ {
		dayStart := today.AddDate(0, 0, -(syntheticDays - 1 - d))
		dayLen := 24 * time.Hour
		if dayEnd := dayStart.Add(dayLen); dayEnd.After(now) {
			// Today is only partially elapsed: spread rows over [dayStart, now)
			// so they don't pile up on a single clamped timestamp.
			dayLen = now.Sub(dayStart)
		}
		for i := 0; i < counts[d]; i++ {
			reversedAt := dayStart.Add(time.Duration(rng.Float64() * float64(dayLen)))
			createdAt := reversedAt.Add(time.Duration(rng.Intn(8*60)+1) * time.Minute)
			if createdAt.After(now) {
				createdAt = reversedAt.Add(time.Duration(rng.Float64() * float64(now.Sub(reversedAt))))
			}

			srcRoll := rng.Float64()
			var src models.Source
			var related *models.SteamID
			switch {
			case srcRoll < 0.90:
				src = models.SourceDirect
			case srcRoll < 0.95:
				src = models.SourceRelatedUser
				relID := models.SteamID(SyntheticSteamIDMin + (steamOffset+10_000)*97)
				related = &relID
			default:
				src = models.SourceUserReport
			}

			var expunged *uint64
			if rng.Float64() < 0.015 {
				eAt := createdAt.Add(time.Duration(rng.Intn(24*60)+1) * time.Minute)
				if eAt.Before(now) {
					ems := uint64(eAt.UnixMilli())
					expunged = &ems
				}
			}

			steamID := models.SteamID(SyntheticSteamIDMin + steamOffset*73 + uint64(rng.Intn(50)))
			steamOffset++

			// Snowflake encodes created_at + a 12-bit per-ms sequence;
			// mirrors domain/models/snowflake.go so generated IDs sort
			// chronologically alongside production rows.
			seq = (seq + 1) & 0x0FFF
			// Clamp so a createdAt before models.Epoch can't underflow the
			// unsigned subtraction (mirrors the guard in genSnowflakeWithParts).
			createdAtMs := uint64(createdAt.UnixMilli())
			if createdAtMs < models.Epoch {
				createdAtMs = models.Epoch
			}
			sf := models.Snowflake(((createdAtMs - models.Epoch) << 22) | uint64(seq))

			rows = append(rows, &models.Reversal{
				Model: models.Model{
					ID:        sf,
					CreatedAt: uint64(createdAt.UnixMilli()),
					UpdatedAt: uint64(createdAt.UnixMilli()),
				},
				SteamID:         steamID,
				MarketplaceSlug: pickMarketplace(rng),
				Source:          &src,
				RelatedSteamID:  related,
				ReversedAt:      uint64(reversedAt.UnixMilli()),
				ExpungedAt:      expunged,
			})
		}
	}
	return rows
}

func pickMarketplace(rng *rand.Rand) string {
	r := rng.Float64()
	cum := 0.0
	for _, mp := range syntheticMarketplaces {
		cum += mp.weight
		if r < cum {
			return mp.slug
		}
	}
	return syntheticMarketplaces[0].slug
}

// insertChunkSize keeps each bulk insert under Postgres's 65,535
// parameter-per-statement cap. At ~11 columns per Reversal, 1,000 rows
// uses ~11k parameters.
const insertChunkSize = 1000

// InsertReversals bulk-inserts reversals, skipping any row whose
// (steam_id, marketplace_slug) already exists via ON CONFLICT DO NOTHING.
// The conflict target matches the partial unique index created in
// repository/public (idx_reversals_steam_id_marketplace_slug ... WHERE
// deleted_at IS NULL). Those pairs don't depend on the clock, so a rerun
// without ResetSynthetic inserts nothing.
//
// Only that index is covered: a primary-key collision on id still fails the
// whole batch. Snowflakes embed created_at plus a sequence, so this would
// require two seeds generating the same millisecond and sequence.
//
// Returns the number of rows actually inserted.
func InsertReversals(db *gorm.DB, reversals []*models.Reversal) (int64, error) {
	if len(reversals) == 0 {
		return 0, nil
	}
	onConflict := clause.OnConflict{
		Columns:     []clause.Column{{Name: "steam_id"}, {Name: "marketplace_slug"}},
		TargetWhere: clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "deleted_at IS NULL"}}},
		DoNothing:   true,
	}
	var inserted int64
	for i := 0; i < len(reversals); i += insertChunkSize {
		end := min(i+insertChunkSize, len(reversals))
		res := db.Clauses(onConflict).Create(reversals[i:end])
		if res.Error != nil {
			return inserted, res.Error
		}
		inserted += res.RowsAffected
	}
	return inserted, nil
}

// ResetSynthetic hard-deletes every reversal whose steam_id falls in the
// synthetic range, including soft-deleted rows. Returns the number of rows
// removed.
func ResetSynthetic(db *gorm.DB) (int64, error) {
	res := db.Unscoped().
		Where("steam_id BETWEEN ? AND ?", SyntheticSteamIDMin, SyntheticSteamIDMax).
		Delete(&models.Reversal{})
	return res.RowsAffected, res.Error
}
