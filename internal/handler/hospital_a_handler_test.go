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
	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type mockHospitalAService struct {
	result *response.HospitalAPatientResponse
	err    error
}

func (m *mockHospitalAService) SearchPatient(_ context.Context, _ string) (*response.HospitalAPatientResponse, error) {
	return m.result, m.err
}

func newHospitalARouter(svc service.HospitalAService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewHospitalAHandler(svc)
	r.GET("/hospital-a/patient/search/:id", h.SearchPatient)
	return r
}

func TestHospitalAHandler_SearchPatient_Success(t *testing.T) {
	natID := "1234567890123"
	svc := &mockHospitalAService{
		result: &response.HospitalAPatientResponse{
			NationalID:  &natID,
			FirstNameTH: "สมชาย",
			LastNameTH:  "ใจดี",
			DateOfBirth: "1990-01-15",
			Gender:      "male",
		},
	}
	r := newHospitalARouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/hospital-a/patient/search/1234567890123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var body response.HospitalAPatientResponse
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if body.NationalID == nil || *body.NationalID != "1234567890123" {
		t.Errorf("expected national_id '1234567890123', got %v", body.NationalID)
	}
	if body.FirstNameTH != "สมชาย" {
		t.Errorf("expected first_name_th 'สมชาย', got %q", body.FirstNameTH)
	}
}

func TestHospitalAHandler_SearchPatient_NullableFieldsNull(t *testing.T) {
	svc := &mockHospitalAService{
		result: &response.HospitalAPatientResponse{
			FirstNameTH: "ทดสอบ",
			LastNameTH:  "ระบบ",
			DateOfBirth: "2000-01-01",
			Gender:      "female",
		},
	}
	r := newHospitalARouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/hospital-a/patient/search/AB123456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var raw map[string]any
	json.Unmarshal(w.Body.Bytes(), &raw) //nolint:errcheck
	if raw["national_id"] != nil {
		t.Errorf("expected national_id null, got %v", raw["national_id"])
	}
	if raw["passport_id"] != nil {
		t.Errorf("expected passport_id null, got %v", raw["passport_id"])
	}
}

func TestHospitalAHandler_SearchPatient_NotFound(t *testing.T) {
	svc := &mockHospitalAService{err: service.ErrHospitalANotFound}
	r := newHospitalARouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/hospital-a/patient/search/unknown", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var body response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &body) //nolint:errcheck
	if body.Error.Code != "NOT_FOUND" {
		t.Errorf("expected code NOT_FOUND, got %q", body.Error.Code)
	}
}

func TestHospitalAHandler_SearchPatient_UpstreamError(t *testing.T) {
	svc := &mockHospitalAService{err: service.ErrHospitalAUpstream}
	r := newHospitalARouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/hospital-a/patient/search/1234567890123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
	var body response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &body) //nolint:errcheck
	if body.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected code UPSTREAM_ERROR, got %q", body.Error.Code)
	}
}

func TestHospitalAHandler_SearchPatient_UnexpectedError_Returns502(t *testing.T) {
	svc := &mockHospitalAService{err: errors.New("something unexpected")}
	r := newHospitalARouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/hospital-a/patient/search/1234567890123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", w.Code)
	}
	var body response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &body) //nolint:errcheck
	if body.Error.Code != "UPSTREAM_ERROR" {
		t.Errorf("expected code UPSTREAM_ERROR, got %q", body.Error.Code)
	}
}
