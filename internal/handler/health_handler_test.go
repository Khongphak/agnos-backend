package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type mockHealthService struct {
	err error
}

func (m *mockHealthService) Check(_ context.Context) error {
	return m.err
}

func TestHealthHandler_GetHealth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewHealthHandler(&mockHealthService{})

	router := gin.New()
	router.GET("/health", h.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp response.HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status %q, got %q", "ok", resp.Status)
	}
	if resp.Database != "connected" {
		t.Errorf("expected database %q, got %q", "connected", resp.Database)
	}
}

func TestHealthHandler_GetHealth_DBUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	h := handler.NewHealthHandler(&mockHealthService{err: errors.New("ping failed")})

	router := gin.New()
	router.GET("/health", h.GetHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", w.Code)
	}

	var resp response.ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response body: %v", err)
	}
	if resp.Error.Code != "DB_UNAVAILABLE" {
		t.Errorf("expected error code %q, got %q", "DB_UNAVAILABLE", resp.Error.Code)
	}
}
