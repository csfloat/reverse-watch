package devseed

import (
	"math"
	"math/rand"
	"time"

	"reverse-watch/domain/models"
)

const (
	syntheticRNGSeed      int64  = 42
	syntheticDays                = 180
	syntheticTargetTotal         = 9800
	syntheticBaseSteamID  uint64 = 76561198000000000
	syntheticBaseReporter uint   = 2_900_000
)

var syntheticMarketplaces = []struct {
	slug   string
	weight float64
}{
	{"csfloat", 0.80},
	{"tradeit", 0.10},
	{"skinport", 0.05},
	{"swap.gg", 0.05},
}

// GenerateSynthetic returns a deterministic ~6-month dataset (~9,800 rows,
// at least one per day, gentle sinusoid with occasional spikes / quiet
// days). Snowflake IDs are unique within the slice and won't collide with
// real CSV-seeded IDs, so callers can pipe the result straight into
// InsertReversals.
func GenerateSynthetic(now time.Time) []*models.Reversal {
	rng := rand.New(rand.NewSource(syntheticRNGSeed))
	nowMs := uint64(now.UnixMilli())
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
		dayStart := today.AddDate(0, 0, -(syntheticDays-1-d))
		for i := 0; i < counts[d]; i++ {
			reversedAt := dayStart.Add(time.Duration(rng.Float64() * float64(24*time.Hour)))
			if uint64(reversedAt.UnixMilli()) > nowMs {
				reversedAt = now.Add(-1 * time.Minute)
			}
			reportDelay := time.Duration(rng.Intn(8*60)+1) * time.Minute
			createdAt := reversedAt.Add(reportDelay)
			if uint64(createdAt.UnixMilli()) > nowMs {
				createdAt = now
			}

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

			var expunged *uint64
			if rng.Float64() < 0.015 {
				eAt := createdAt.Add(time.Duration(rng.Intn(24*60)+1) * time.Minute)
				if uint64(eAt.UnixMilli()) < nowMs && uint64(eAt.UnixMilli()) > uint64(createdAt.UnixMilli()) {
					ems := uint64(eAt.UnixMilli())
					expunged = &ems
				}
			}

			steamID := models.SteamID(syntheticBaseSteamID + steamOffset*73 + uint64(rng.Intn(50)))
			steamOffset++

			// Snowflake encodes created_at + a 12-bit per-ms sequence;
			// mirrors domain/models/snowflake.go so generated IDs sort
			// chronologically alongside production rows.
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
