package database_test

import (
	"errors"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/database"
)

func TestRetryPing_SucceedsOnFirstAttempt(t *testing.T) {
	calls := 0
	ping := func() error {
		calls++
		return nil
	}

	err := database.RetryPing(ping, database.RetryConfig{MaxAttempts: 3, RetryInterval: 0})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 ping call, got %d", calls)
	}
}

func TestRetryPing_RetriesUntilSuccess(t *testing.T) {
	calls := 0
	ping := func() error {
		calls++
		if calls < 3 {
			return errors.New("not ready")
		}
		return nil
	}

	err := database.RetryPing(ping, database.RetryConfig{MaxAttempts: 5, RetryInterval: 0})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 ping calls, got %d", calls)
	}
}

func TestRetryPing_FailsAfterMaxAttempts(t *testing.T) {
	calls := 0
	sentinel := errors.New("connection refused")
	ping := func() error {
		calls++
		return sentinel
	}

	err := database.RetryPing(ping, database.RetryConfig{MaxAttempts: 3, RetryInterval: 0})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 3 {
		t.Errorf("expected 3 ping calls, got %d", calls)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected wrapped sentinel error, got %v", err)
	}
}

func TestRetryPing_ZeroAttempts(t *testing.T) {
	ping := func() error { return nil }

	err := database.RetryPing(ping, database.RetryConfig{MaxAttempts: 0, RetryInterval: 0})
	if err == nil {
		t.Fatal("expected error for zero attempts, got nil")
	}
}
