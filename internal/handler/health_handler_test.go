package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type mockHealthService struct {
	status   string
	database string
}

func (m *mockHealthService) Check() (string, string) {
	return m.status, m.database
}

func TestHealthHandler_GetHealth_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockHealthService{status: "ok", database: "connected"}
	h := handler.NewHealthHandler(svc)

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

func TestHealthHandler_GetHealth_DBDisconnected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &mockHealthService{status: "ok", database: "disconnected"}
	h := handler.NewHealthHandler(svc)

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
	if resp.Database != "disconnected" {
		t.Errorf("expected database %q, got %q", "disconnected", resp.Database)
	}
}
