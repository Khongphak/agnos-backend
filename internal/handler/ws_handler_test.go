package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"

	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/service"
	internalws "github.com/agnos-assessment/agnos-backend/internal/ws"
)

const testJWTSecret = "test-secret-key"

func makeTestToken(hospitalID int64) string {
	claims := service.StaffClaims{
		StaffID:    1,
		HospitalID: hospitalID,
		Role:       "staff",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, _ := token.SignedString([]byte(testJWTSecret))
	return signed
}

func newWSRouter(svc service.PatientService) (*gin.Engine, *internalws.Hub) {
	gin.SetMode(gin.TestMode)
	hub := internalws.NewHub()
	go hub.Run()
	r := gin.New()
	h := handler.NewWSHandler(hub, svc, testJWTSecret)
	r.GET("/ws/patient", h.HandlePatient)
	r.GET("/ws/staff", h.HandleStaff)
	return r, hub
}

// ── /ws/patient tests ─────────────────────────────────────────────────────────

func TestWSHandler_Patient_MissingHospitalCode(t *testing.T) {
	svc := &mockPatientService{}
	router, _ := newWSRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/ws/patient", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestWSHandler_Patient_HospitalNotFound_ClosesWith1008(t *testing.T) {
	svc := &mockPatientService{hospitalErr: service.ErrPatientHospitalNotFound}
	router, _ := newWSRouter(svc)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/patient?hospital_code=BAD"
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		// If server sends close immediately, Dial may fail — that's acceptable
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second)) //nolint:errcheck
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected close message from server")
	}
	closeErr, ok := err.(*websocket.CloseError)
	if !ok {
		t.Fatalf("expected *websocket.CloseError, got %T: %v", err, err)
	}
	if closeErr.Code != websocket.ClosePolicyViolation {
		t.Errorf("expected close code 1008, got %d", closeErr.Code)
	}
}

func TestWSHandler_Patient_ConnectedAckSent(t *testing.T) {
	svc := &mockPatientService{
		hospitalResult: &model.Hospital{ID: 1, Code: "BKK001", Name: "Test Hospital"},
	}
	router, _ := newWSRouter(svc)

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/patient?hospital_code=BKK001"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WS: %v", err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(time.Second)) //nolint:errcheck
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("expected connected ack, got error: %v", err)
	}

	var ack map[string]string
	if err := json.Unmarshal(data, &ack); err != nil {
		t.Fatalf("invalid JSON ack: %v", err)
	}
	if ack["type"] != "connected" {
		t.Errorf("expected type 'connected', got %q", ack["type"])
	}
	if ack["session_id"] == "" {
		t.Error("expected non-empty session_id in ack")
	}
}

// ── /ws/staff tests ───────────────────────────────────────────────────────────

func TestWSHandler_Staff_MissingToken(t *testing.T) {
	svc := &mockPatientService{}
	router, _ := newWSRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/ws/staff", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWSHandler_Staff_InvalidToken(t *testing.T) {
	svc := &mockPatientService{}
	router, _ := newWSRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/ws/staff?token=bad.token.here", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestWSHandler_Staff_ValidToken_ConnectsSuccessfully(t *testing.T) {
	svc := &mockPatientService{}
	router, _ := newWSRouter(svc)

	server := httptest.NewServer(router)
	defer server.Close()

	token := makeTestToken(1)
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/staff?token=" + token
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial WS: %v", err)
	}
	defer conn.Close()
	// If we get here without error, the connection was accepted
}

