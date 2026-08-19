package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/handler"
	"github.com/agnos-assessment/agnos-backend/internal/middleware"
	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

// ── mock PatientService ───────────────────────────────────────────────────────

type mockPatientService struct {
	searchResult   []model.Patient
	searchErr      error
	hospitalResult *model.Hospital
	hospitalErr    error
}

func (m *mockPatientService) Search(_ context.Context, _ service.SearchParams) ([]model.Patient, error) {
	return m.searchResult, m.searchErr
}

func (m *mockPatientService) FindHospitalByCode(_ context.Context, _ string) (*model.Hospital, error) {
	return m.hospitalResult, m.hospitalErr
}

// ── router helper ─────────────────────────────────────────────────────────────

func newPatientRouter(svc service.PatientService, claims *service.StaffClaims) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := handler.NewPatientHandler(svc)
	r.GET("/patient/search", func(c *gin.Context) {
		c.Set(middleware.StaffClaimsKey, claims)
		h.Search(c)
	})
	return r
}

func getRequest(router *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── tests ─────────────────────────────────────────────────────────────────────

var defaultClaims = &service.StaffClaims{StaffID: 1, HospitalID: 42, Role: "staff"}

func TestPatientHandler_Search_EmptyResultReturnsEmptyList(t *testing.T) {
	svc := &mockPatientService{searchResult: nil}
	w := getRequest(newPatientRouter(svc, defaultClaims), "/patient/search")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var body response.PatientListResponse
	json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Patients) != 0 {
		t.Errorf("expected empty patients list, got %d items", len(body.Patients))
	}
}

func TestPatientHandler_Search_ReturnsPatients(t *testing.T) {
	natID := "1234567890123"
	dob, _ := time.Parse("2006-01-02", "1990-01-15")
	svc := &mockPatientService{
		searchResult: []model.Patient{
			{
				ID:          1,
				NationalID:  &natID,
				FirstNameTH: "สมชาย",
				LastNameTH:  "ใจดี",
				DateOfBirth: dob,
				Gender:      "male",
				PatientHN:   "HN001",
			},
		},
	}
	w := getRequest(newPatientRouter(svc, defaultClaims), "/patient/search")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var body response.PatientListResponse
	json.Unmarshal(w.Body.Bytes(), &body)
	if len(body.Patients) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(body.Patients))
	}
	p := body.Patients[0]
	if p.ID != 1 {
		t.Errorf("expected id=1, got %d", p.ID)
	}
	if p.PatientHN != "HN001" {
		t.Errorf("expected patient_hn 'HN001', got %q", p.PatientHN)
	}
	if p.DateOfBirth != "1990-01-15" {
		t.Errorf("expected date_of_birth '1990-01-15', got %q", p.DateOfBirth)
	}
}

func TestPatientHandler_Search_InternalError(t *testing.T) {
	svc := &mockPatientService{searchErr: errors.New("db down")}
	w := getRequest(newPatientRouter(svc, defaultClaims), "/patient/search")
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	var body response.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &body)
	if body.Error.Code != "INTERNAL_ERROR" {
		t.Errorf("expected INTERNAL_ERROR, got %q", body.Error.Code)
	}
}

func TestPatientHandler_Search_NullableFieldsHandled(t *testing.T) {
	dob, _ := time.Parse("2006-01-02", "2000-06-15")
	svc := &mockPatientService{
		searchResult: []model.Patient{
			{
				ID:          2,
				NationalID:  nil,
				PassportID:  nil,
				FirstNameTH: "นามสมมุติ",
				LastNameTH:  "ทดสอบ",
				DateOfBirth: dob,
				Gender:      "female",
				PatientHN:   "HN002",
			},
		},
	}
	w := getRequest(newPatientRouter(svc, defaultClaims), "/patient/search")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body)
	}
	var raw map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &raw)
	patients := raw["patients"].([]interface{})
	p := patients[0].(map[string]interface{})
	if p["national_id"] != nil {
		t.Errorf("expected national_id null, got %v", p["national_id"])
	}
}
