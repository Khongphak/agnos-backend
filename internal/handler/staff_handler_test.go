package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type mockStaffService struct {
	createResult *service.CreateStaffResponse
	createErr    error
	loginResult  *service.LoginResponse
	loginErr     error
}

func (m *mockStaffService) Create(_ context.Context, _ service.CreateStaffRequest) (*service.CreateStaffResponse, error) {
	return m.createResult, m.createErr
}

func (m *mockStaffService) Login(_ context.Context, _ service.LoginRequest) (*service.LoginResponse, error) {
	return m.loginResult, m.loginErr
}

func newStaffRouter(svc service.StaffService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewStaffHandler(svc)
	r.POST("/staff/create", h.Create)
	r.POST("/staff/login", h.Login)
	return r
}

func postJSON(router *gin.Engine, path string, body interface{}) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── POST /staff/create ───────────────────────────────────────────────────────

func TestStaffHandler_Create_Success(t *testing.T) {
	svc := &mockStaffService{
		createResult: &service.CreateStaffResponse{
			ID:           1,
			Username:     "alice",
			HospitalCode: "H1",
			Role:         "staff",
		},
	}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username":      "alice",
		"password":      "pass123",
		"hospital_code": "H1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["username"] != "alice" {
		t.Errorf("expected username 'alice', got %v", body["username"])
	}
	if body["role"] != "staff" {
		t.Errorf("expected role 'staff', got %v", body["role"])
	}
}

func TestStaffHandler_Create_DefaultRoleIsStaff(t *testing.T) {
	svc := &mockStaffService{
		createResult: &service.CreateStaffResponse{ID: 1, Username: "u", HospitalCode: "H1", Role: "staff"},
	}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username":      "u",
		"password":      "p",
		"hospital_code": "H1",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestStaffHandler_Create_AdminRole(t *testing.T) {
	svc := &mockStaffService{
		createResult: &service.CreateStaffResponse{ID: 2, Username: "admin1", HospitalCode: "H1", Role: "admin"},
	}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username":      "admin1",
		"password":      "pass",
		"hospital_code": "H1",
		"role":          "admin",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestStaffHandler_Create_InvalidRole(t *testing.T) {
	svc := &mockStaffService{}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username":      "u",
		"password":      "p",
		"hospital_code": "H1",
		"role":          "superuser",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "INVALID_INPUT" {
		t.Errorf("expected INVALID_INPUT, got %q", resp.Error.Code)
	}
}

func TestStaffHandler_Create_MissingFields(t *testing.T) {
	svc := &mockStaffService{}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{"username": "u"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStaffHandler_Create_HospitalNotFound(t *testing.T) {
	svc := &mockStaffService{createErr: service.ErrHospitalNotFound}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username": "u", "password": "p", "hospital_code": "BAD",
	})
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "HOSPITAL_NOT_FOUND" {
		t.Errorf("expected HOSPITAL_NOT_FOUND, got %q", resp.Error.Code)
	}
}

func TestStaffHandler_Create_UsernameConflict(t *testing.T) {
	svc := &mockStaffService{createErr: service.ErrUsernameConflict}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username": "u", "password": "p", "hospital_code": "H1",
	})
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var resp response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "USERNAME_CONFLICT" {
		t.Errorf("expected USERNAME_CONFLICT, got %q", resp.Error.Code)
	}
}

func TestStaffHandler_Create_InternalError(t *testing.T) {
	svc := &mockStaffService{createErr: errors.New("db down")}
	w := postJSON(newStaffRouter(svc), "/staff/create", map[string]string{
		"username": "u", "password": "p", "hospital_code": "H1",
	})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ── POST /staff/login ────────────────────────────────────────────────────────

func TestStaffHandler_Login_Success(t *testing.T) {
	svc := &mockStaffService{
		loginResult: &service.LoginResponse{
			AccessToken:  "access.token.here",
			RefreshToken: "refresh-token-raw",
			ExpiresIn:    900,
		},
	}
	w := postJSON(newStaffRouter(svc), "/staff/login", map[string]string{
		"username": "bob", "password": "pass", "hospital_code": "H1",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["access_token"] != "access.token.here" {
		t.Errorf("unexpected access_token: %v", body["access_token"])
	}
	if body["refresh_token"] != "refresh-token-raw" {
		t.Errorf("unexpected refresh_token: %v", body["refresh_token"])
	}
	if body["expires_in"] != float64(900) {
		t.Errorf("unexpected expires_in: %v", body["expires_in"])
	}
}

func TestStaffHandler_Login_MissingFields(t *testing.T) {
	svc := &mockStaffService{}
	w := postJSON(newStaffRouter(svc), "/staff/login", map[string]string{"username": "bob"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestStaffHandler_Login_InvalidCredentials(t *testing.T) {
	svc := &mockStaffService{loginErr: service.ErrInvalidCredentials}
	w := postJSON(newStaffRouter(svc), "/staff/login", map[string]string{
		"username": "bob", "password": "wrong", "hospital_code": "H1",
	})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
	var resp response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "INVALID_CREDENTIALS" {
		t.Errorf("expected INVALID_CREDENTIALS, got %q", resp.Error.Code)
	}
}

func TestStaffHandler_Login_AccountInactive(t *testing.T) {
	svc := &mockStaffService{loginErr: service.ErrAccountInactive}
	w := postJSON(newStaffRouter(svc), "/staff/login", map[string]string{
		"username": "bob", "password": "pass", "hospital_code": "H1",
	})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	var resp response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error.Code != "ACCOUNT_INACTIVE" {
		t.Errorf("expected ACCOUNT_INACTIVE, got %q", resp.Error.Code)
	}
}

func TestStaffHandler_Login_InternalError(t *testing.T) {
	svc := &mockStaffService{loginErr: errors.New("unexpected")}
	w := postJSON(newStaffRouter(svc), "/staff/login", map[string]string{
		"username": "bob", "password": "pass", "hospital_code": "H1",
	})
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
