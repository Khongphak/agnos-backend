package database

import (
	"errors"
	"testing"

	"github.com/golang-migrate/migrate/v4"
)

// mockMigrator satisfies the migrator interface for unit testing without a real DB.
type mockMigrator struct {
	upErr      error
	version    uint
	versionErr error
}

func (m *mockMigrator) Up() error                    { return m.upErr }
func (m *mockMigrator) Version() (uint, bool, error) { return m.version, false, m.versionErr }
func (m *mockMigrator) Close() (error, error)        { return nil, nil }

func TestApplyMigrations_NoChange(t *testing.T) {
	err := applyMigrations(&mockMigrator{upErr: migrate.ErrNoChange})
	if err != nil {
		t.Fatalf("ErrNoChange should return nil, got %v", err)
	}
}

func TestApplyMigrations_UpError(t *testing.T) {
	sentinel := errors.New("dirty migration state")
	err := applyMigrations(&mockMigrator{upErr: sentinel})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

func TestApplyMigrations_Success(t *testing.T) {
	err := applyMigrations(&mockMigrator{version: 3})
	if err != nil {
		t.Fatalf("expected nil on successful migration, got %v", err)
	}
}

func TestApplyMigrations_SuccessVersionUnknown(t *testing.T) {
	// Version() error after a successful Up() must still return nil to the caller.
	err := applyMigrations(&mockMigrator{versionErr: errors.New("no version recorded")})
	if err != nil {
		t.Fatalf("expected nil when Version() fails post-Up, got %v", err)
	}
}
