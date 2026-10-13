// Command seed loads a synthetic dataset into the local Reverse Watch
// Postgres databases for local dashboard development. It is NEVER intended
// to run in production: it refuses to run unless Config.Environment is
// "development" and Database.Host points at a local database.
//
//	go run ./cmd/seed          # insert; rows that already exist are skipped
//	go run ./cmd/seed -reset   # delete existing synthetic rows first, then insert
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"reverse-watch/config"
	"reverse-watch/domain/models"
	"reverse-watch/domain/models/constants"
	"reverse-watch/internal/devseed"
	"reverse-watch/logging"
	"reverse-watch/repository/factory"
	"reverse-watch/secret"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	reset := flag.Bool("reset", false, "delete existing synthetic reversals before seeding, so a rerun refreshes the data to end at today")
	flag.Parse()

	if err := run(*reset); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(reset bool) error {
	logging.Initialize()
	cfg := config.Load()

	if cfg.Environment != constants.EnvironmentDevelopment {
		return fmt.Errorf("refusing to seed: environment is %q (only %q is allowed)", cfg.Environment, constants.EnvironmentDevelopment)
	}
	if !isLocalHost(cfg.Database.Host) {
		return fmt.Errorf("refusing to seed: database host %q is not local (expected localhost, a loopback address, a unix socket, or a docker service name)", cfg.Database.Host)
	}

	// Required by factory bootstrap (e.g. admin API key seeding). The
	// synthetic generator pre-populates its own IDs, so the snowflake
	// generator does not actually run for them.
	models.InitSnowflakeGenerator(0, 0)

	keygen := secret.NewKeyGenerator(cfg.Environment)
	f, err := factory.NewFactory(cfg, keygen)
	if err != nil {
		return fmt.Errorf("failed to create factory: %w", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close factory: %v\n", err)
		}
	}()

	// The factory logs every statement at Info level, which would dump the
	// full SQL of each 1,000-row batch.
	db := f.PublicDB().Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Warn)})

	if reset {
		deleted, err := devseed.ResetSynthetic(db)
		if err != nil {
			return fmt.Errorf("failed to reset synthetic reversals: %w", err)
		}
		fmt.Printf("reset: deleted %d existing synthetic reversals\n", deleted)
	}

	reversals := devseed.GenerateSynthetic(time.Now().UTC())
	fmt.Printf("generated %d synthetic reversals\n", len(reversals))

	inserted, err := devseed.InsertReversals(db, reversals)
	if err != nil {
		return fmt.Errorf("failed to insert reversals: %w", err)
	}
	skipped := int64(len(reversals)) - inserted
	fmt.Printf("seed complete: %d inserted, %d already present (skipped)\n", inserted, skipped)
	if skipped > 0 && !reset {
		fmt.Println("hint: run with -reset to replace existing synthetic data with a fresh dataset ending today")
	}
	return nil
}

// isLocalHost accepts loopback hosts, unix socket directories, and
// single-label hostnames (docker compose service names such as "postgres").
// Anything else with a dot or colon (FQDNs, non-loopback IPs, Cloud SQL
// connection names like "project:region:instance") is treated as remote.
func isLocalHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || host == "localhost" || strings.HasPrefix(host, "/") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return !strings.ContainsAny(host, ".:")
}
