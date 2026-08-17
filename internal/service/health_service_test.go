package service_test

import (
	"errors"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/service"
)

type mockHealthRepo struct {
	pingErr error
}

func (m *mockHealthRepo) Ping() error {
	return m.pingErr
}

func TestHealthService_Check_DBConnected(t *testing.T) {
	svc := service.NewHealthService(&mockHealthRepo{})
	status, db := svc.Check()
	if status != "ok" {
		t.Errorf("expected status %q, got %q", "ok", status)
	}
	if db != "connected" {
		t.Errorf("expected database %q, got %q", "connected", db)
	}
}

func TestHealthService_Check_DBDisconnected(t *testing.T) {
	svc := service.NewHealthService(&mockHealthRepo{pingErr: errors.New("connection refused")})
	status, db := svc.Check()
	if status != "ok" {
		t.Errorf("expected status %q, got %q", "ok", status)
	}
	if db != "disconnected" {
		t.Errorf("expected database %q, got %q", "disconnected", db)
	}
}
