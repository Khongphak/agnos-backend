package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"

	"github.com/agnos-assessment/agnos-backend/internal/config"
)

// RetryConfig controls how many times and how often Connect retries Ping.
type RetryConfig struct {
	MaxAttempts   int
	RetryInterval time.Duration
}

// DefaultRetryConfig is suitable for container startup (up to ~20 s total wait).
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:   10,
	RetryInterval: 2 * time.Second,
}

// RetryPing calls ping repeatedly until it succeeds or MaxAttempts is exhausted.
// It logs each attempt and sleeps RetryInterval between failures.
// Exported so it can be unit-tested with a mock ping function.
func RetryPing(ping func() error, cfg RetryConfig) error {
	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ping(); err == nil {
			log.Printf("database: connected (attempt %d/%d)", attempt, cfg.MaxAttempts)
			return nil
		} else {
			lastErr = err
			log.Printf("database: ping attempt %d/%d failed: %v", attempt, cfg.MaxAttempts, err)
		}
		if attempt < cfg.MaxAttempts {
			time.Sleep(cfg.RetryInterval)
		}
	}
	return fmt.Errorf("database unreachable after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// Connect opens a postgres connection pool and verifies reachability via RetryPing.
// Returns a ready *sql.DB or a non-nil error if all retries fail.
func Connect(cfg config.DatabaseConfig, retry RetryConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("database: open: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := RetryPing(db.Ping, retry); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
