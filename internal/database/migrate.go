package database

import (
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// migrator is the subset of migrate.Migrate used by applyMigrations.
// Kept unexported so it is only visible within the package (including internal tests).
type migrator interface {
	Up() error
	Version() (uint, bool, error)
	Close() (error, error)
}

// RunMigrations runs all pending SQL migrations from migrationsPath against dsn.
// dsn must be a postgres URL: postgres://user:pass@host:port/dbname?sslmode=...
// Returns nil when migrations are already up to date (ErrNoChange is treated as success).
func RunMigrations(dsn string, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, dsn)
	if err != nil {
		return fmt.Errorf("migrations: init: %w", err)
	}
	return applyMigrations(m)
}

// applyMigrations is the testable core; it accepts the migrator interface so tests
// can inject a mock without hitting a real database.
func applyMigrations(m migrator) error {
	defer m.Close() //nolint:errcheck

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("migrations: already up to date")
			return nil
		}
		return fmt.Errorf("migrations: up: %w", err)
	}

	version, _, err := m.Version()
	if err != nil {
		log.Println("migrations: applied successfully")
		return nil
	}
	log.Printf("migrations: applied version %d", version)
	return nil
}
