// Command seed loads dev-only fixture data into the local Reverse Watch
// Postgres databases. It is NEVER intended to run in production.
//
// Two modes:
//
//	go run ./cmd/seed                  # load the 98-row real CSV fixture
//	go run ./cmd/seed -csv path/.csv   # load a different CSV
//	go run ./cmd/seed -synthetic       # generate a deterministic ~9.8k-row
//	                                   # 6-month dataset with spikes
//
// Run from the repo root. Both modes are idempotent — the underlying
// INSERT uses ON CONFLICT (id) DO NOTHING.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/devseed"
	"reverse-watch/logging"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"
)

func main() {
	csvPath := flag.String("csv", "internal/devseed/fixtures/reversals_seed.csv", "Path to seed CSV file (ignored when -synthetic is set)")
	synthetic := flag.Bool("synthetic", false, "Generate a deterministic 6-month synthetic dataset instead of loading the CSV")
	flag.Parse()

	logging.Initialize()
	cfg := config.Load()

	if cfg.Environment != constants.EnvironmentDevelopment {
		fmt.Fprintf(os.Stderr, "refusing to seed: environment is %q (only %q is allowed)\n", cfg.Environment, constants.EnvironmentDevelopment)
		os.Exit(1)
	}

	// Required by factory bootstrap (e.g. admin API key seeding). Our
	// own seed rows pre-populate their IDs, so the generator does not
	// actually run for them.
	models.InitSnowflakeGenerator(0, 0)

	keygen := secret.NewKeyGenerator(cfg.Environment)
	f, err := factory.NewFactory(cfg, keygen)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create factory: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close factory: %v\n", err)
		}
	}()

	var reversals []*models.Reversal
	if *synthetic {
		reversals = devseed.GenerateSynthetic(time.Now().UTC())
		fmt.Printf("generated %d synthetic reversals (deterministic seed)\n", len(reversals))
	} else {
		reversals, err = devseed.LoadFromCSV(*csvPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to load CSV: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("loaded %d reversals from %s\n", len(reversals), *csvPath)
	}

	inserted, err := devseed.InsertReversals(f.PublicDB(), reversals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to insert reversals: %v\n", err)
		os.Exit(1)
	}
	skipped := int64(len(reversals)) - inserted
	fmt.Printf("seed complete: %d inserted, %d already present (skipped)\n", inserted, skipped)
}
