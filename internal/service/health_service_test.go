package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/service"
)

type mockHealthRepo struct {
	pingErr error
}

func (m *mockHealthRepo) Ping(_ context.Context) error {
	return m.pingErr
}

func TestHealthService_Check_DBConnected(t *testing.T) {
	svc := service.NewHealthService(&mockHealthRepo{})
	if err := svc.Check(context.Background()); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHealthService_Check_DBDisconnected(t *testing.T) {
	svc := service.NewHealthService(&mockHealthRepo{pingErr: errors.New("connection refused")})
	if err := svc.Check(context.Background()); err == nil {
		t.Error("expected non-nil error, got nil")
	}
}
