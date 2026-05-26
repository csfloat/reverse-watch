package devseed

import (
	"math"
	"math/rand"
	"time"

	"reverse-watch/domain/models"
)

// Synthetic data parameters. Tuned to give the dashboard a realistic
// 6-month spread to look at without dwarfing the local DB.
const (
	syntheticRNGSeed     int64  = 42
	syntheticDays               = 180
	syntheticTargetTotal        = 9800
	syntheticBaseSteamID uint64 = 76561198000000000
	syntheticBaseReporter uint  = 2_900_000
)

// Weighted marketplace mix. csfloat dominates to match the production
// reality where it's the only source today; the rest is sprinkled in so
// the table view exercises the JS fallback for unknown slugs.
var syntheticMarketplaces = []struct {
	slug   string
	weight float64
}{
	{"csfloat", 0.80},
	{"tradeit", 0.10},
	{"skinport", 0.05},
	{"swap.gg", 0.05},
}

// GenerateSynthetic builds a deterministic, ~6-month synthetic dataset.
// Same `now` and same code -> same rows. The output is suitable for
// piping straight into InsertReversals (snowflake IDs are unique within
// the slice and won't collide with real CSV-seeded IDs).
//
// Shape of the result:
//   - Spans (now - 180 UTC days) through `now`, inclusive of every day.
//   - ~9,800 rows total, with at least 1 reversal on every day.
//   - Daily counts follow a gentle sinusoid around ~50/day with ~5%
//     spike days (2.5-5x), ~10% quiet days (0.2-0.5x), rest normal.
//   - 80% csfloat / 10% tradeit / 5% skinport / 5% swap.gg.
//   - 90% source=direct, 5% related_user (with valid related_steam_id),
//     5% user_report.
//   - ~1.5% rows are expunged (non-nil ExpungedAt).
func GenerateSynthetic(now time.Time) []*models.Reversal {
	rng := rand.New(rand.NewSource(syntheticRNGSeed))
	nowMs := uint64(now.UnixMilli())

	// Anchor "today" at UTC midnight so the rightmost bucket on the chart
	// is the current day.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Phase 1: pick a per-day target count.
	counts := make([]int, syntheticDays)
	for d := 0; d < syntheticDays; d++ {
		base := 40.0 + 10.0*math.Sin(float64(d)/30.0)
		var mult float64
		switch r := rng.Float64(); {
		case r < 0.05:
			mult = 2.5 + rng.Float64()*2.5 // spike: 2.5-5x
		case r < 0.15:
			mult = 0.2 + rng.Float64()*0.3 // quiet: 0.2-0.5x
		default:
			mult = 0.7 + rng.Float64()*0.6 // normal: 0.7-1.3x
		}
		counts[d] = int(math.Max(1, base*mult+rng.NormFloat64()*5))
	}

	// Phase 2: scale to hit the target total without dropping below 1/day.
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

	// Phase 3: emit rows.
	rows := make([]*models.Reversal, 0, syntheticTargetTotal+200)
	var steamOffset uint64 = 1
	var seq uint16

	for d := 0; d < syntheticDays; d++ {
		dayStart := today.AddDate(0, 0, -(syntheticDays-1-d))
		for i := 0; i < counts[d]; i++ {
			// reversed_at: random instant inside the day, clamped <= now.
			reversedAt := dayStart.Add(time.Duration(rng.Float64() * float64(24*time.Hour)))
			if uint64(reversedAt.UnixMilli()) > nowMs {
				reversedAt = now.Add(-1 * time.Minute)
			}
			// created_at: 1 minute to 8 hours after reversed_at (the
			// reporter discovers it and submits), clamped <= now.
			reportDelay := time.Duration(rng.Intn(8*60)+1) * time.Minute
			createdAt := reversedAt.Add(reportDelay)
			if uint64(createdAt.UnixMilli()) > nowMs {
				createdAt = now
			}

			// Source distribution + related_steam_id pairing.
			srcRoll := rng.Float64()
			var src models.Source
			var related *models.SteamID
			switch {
			case srcRoll < 0.90:
				src = models.SourceDirect
			case srcRoll < 0.95:
				src = models.SourceRelatedUser
				relID := models.SteamID(syntheticBaseSteamID + (steamOffset+10_000)*97)
				related = &relID
			default:
				src = models.SourceUserReport
			}

			// ~1.5% rows expunge between 1m and 24h after the report.
			var expunged *uint64
			if rng.Float64() < 0.015 {
				eAt := createdAt.Add(time.Duration(rng.Intn(24*60)+1) * time.Minute)
				if uint64(eAt.UnixMilli()) < nowMs && uint64(eAt.UnixMilli()) > uint64(createdAt.UnixMilli()) {
					ems := uint64(eAt.UnixMilli())
					expunged = &ems
				}
			}

			// Each row gets a unique steam_id derived from a monotonically
			// increasing offset (collision-free) with mild jitter.
			steamID := models.SteamID(syntheticBaseSteamID + steamOffset*73 + uint64(rng.Intn(50)))
			steamOffset++

			// Snowflake encodes the row's created_at timestamp + a 12-bit
			// sequence for uniqueness within a millisecond. Mirrors the
			// layout in domain/models/snowflake.go.
			seq = (seq + 1) & 0x0FFF
			sfTs := uint64(createdAt.UnixMilli()) - models.Epoch
			sf := models.Snowflake((sfTs << 22) | uint64(seq))

			reporter := syntheticBaseReporter + uint(steamOffset)
			mp := pickMarketplace(rng)

			rows = append(rows, &models.Reversal{
				Model: models.Model{
					ID:        sf,
					CreatedAt: uint64(createdAt.UnixMilli()),
					UpdatedAt: uint64(createdAt.UnixMilli()),
				},
				SteamID:            steamID,
				MarketplaceSlug:    mp,
				Source:             &src,
				RelatedSteamID:     related,
				ReversedAt:         uint64(reversedAt.UnixMilli()),
				ReporterInternalID: &reporter,
				ExpungedAt:         expunged,
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
