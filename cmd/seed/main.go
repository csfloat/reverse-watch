// Command seed loads a deterministic synthetic dataset into the local
// Reverse Watch Postgres databases for local dashboard development.
// It is NEVER intended to run in production and refuses to run unless
// Config.Environment is "development".
//
//	go run ./cmd/seed
//
// The insert uses ON CONFLICT (id) DO NOTHING, so re-running is safe.
package main

import (
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
	logging.Initialize()
	cfg := config.Load()

	if cfg.Environment != constants.EnvironmentDevelopment {
		fmt.Fprintf(os.Stderr, "refusing to seed: environment is %q (only %q is allowed)\n", cfg.Environment, constants.EnvironmentDevelopment)
		os.Exit(1)
	}

	// Required by factory bootstrap (e.g. admin API key seeding). The
	// synthetic generator pre-populates its own IDs, so the snowflake
	// generator does not actually run for them.
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

	reversals := devseed.GenerateSynthetic(time.Now().UTC())
	fmt.Printf("generated %d synthetic reversals (deterministic seed)\n", len(reversals))

	inserted, err := devseed.InsertReversals(f.PublicDB(), reversals)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to insert reversals: %v\n", err)
		os.Exit(1)
	}
	skipped := int64(len(reversals)) - inserted
	fmt.Printf("seed complete: %d inserted, %d already present (skipped)\n", inserted, skipped)
}
