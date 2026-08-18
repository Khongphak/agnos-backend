package repository_test

import (
	"errors"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/repository"
)

// Verify sentinel errors are distinct and non-nil.
func TestStaffRepositoryErrors_Distinct(t *testing.T) {
	if repository.ErrNotFound == nil {
		t.Fatal("ErrNotFound must not be nil")
	}
	if repository.ErrDuplicateUsername == nil {
		t.Fatal("ErrDuplicateUsername must not be nil")
	}
	if errors.Is(repository.ErrNotFound, repository.ErrDuplicateUsername) {
		t.Error("ErrNotFound and ErrDuplicateUsername must be distinct sentinel errors")
	}
}

// Verify wrapping: a wrapped ErrNotFound is still detected by errors.Is.
func TestStaffRepositoryErrors_Wrapping(t *testing.T) {
	wrapped := errors.Join(errors.New("query failed"), repository.ErrNotFound)
	if !errors.Is(wrapped, repository.ErrNotFound) {
		t.Error("errors.Is must detect ErrNotFound through errors.Join")
	}
}
