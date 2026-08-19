package service_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/service"
)

func newUpstreamServer(status int, body any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != nil {
			json.NewEncoder(w).Encode(body) //nolint:errcheck
		}
	}))
}

func TestHospitalAService_SearchPatient_Success(t *testing.T) {
	upstream := newUpstreamServer(http.StatusOK, map[string]any{
		"national_id":   "1234567890123",
		"first_name_th": "สมชาย",
		"last_name_th":  "ใจดี",
		"first_name_en": "Somchai",
		"last_name_en":  "Jaidee",
		"date_of_birth": "1990-01-15",
		"gender":        "male",
		"phone_number":  "0812345678",
		"email":         "somchai@example.com",
	})
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	p, err := svc.SearchPatient(context.Background(), "1234567890123")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if p.NationalID == nil || *p.NationalID != "1234567890123" {
		t.Errorf("expected national_id '1234567890123', got %v", p.NationalID)
	}
	if p.FirstNameTH != "สมชาย" {
		t.Errorf("expected first_name_th 'สมชาย', got %q", p.FirstNameTH)
	}
	if p.LastNameTH != "ใจดี" {
		t.Errorf("expected last_name_th 'ใจดี', got %q", p.LastNameTH)
	}
	if p.FirstNameEN == nil || *p.FirstNameEN != "Somchai" {
		t.Errorf("expected first_name_en 'Somchai', got %v", p.FirstNameEN)
	}
	if p.Gender != "male" {
		t.Errorf("expected gender 'male', got %q", p.Gender)
	}
	if p.PhoneNumber == nil || *p.PhoneNumber != "0812345678" {
		t.Errorf("expected phone_number '0812345678', got %v", p.PhoneNumber)
	}
	if p.Email == nil || *p.Email != "somchai@example.com" {
		t.Errorf("expected email 'somchai@example.com', got %v", p.Email)
	}
}

func TestHospitalAService_SearchPatient_NullableFieldsAbsent(t *testing.T) {
	upstream := newUpstreamServer(http.StatusOK, map[string]any{
		"national_id":   "",
		"passport_id":   "",
		"first_name_th": "ทดสอบ",
		"last_name_th":  "ระบบ",
		"first_name_en": "",
		"last_name_en":  "",
		"date_of_birth": "2000-01-01",
		"gender":        "female",
		"phone_number":  "",
		"email":         "",
	})
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	p, err := svc.SearchPatient(context.Background(), "AB123456")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if p.NationalID != nil {
		t.Errorf("expected national_id nil, got %v", p.NationalID)
	}
	if p.PassportID != nil {
		t.Errorf("expected passport_id nil, got %v", p.PassportID)
	}
	if p.FirstNameEN != nil {
		t.Errorf("expected first_name_en nil, got %v", p.FirstNameEN)
	}
	if p.LastNameEN != nil {
		t.Errorf("expected last_name_en nil, got %v", p.LastNameEN)
	}
	if p.PhoneNumber != nil {
		t.Errorf("expected phone_number nil, got %v", p.PhoneNumber)
	}
	if p.Email != nil {
		t.Errorf("expected email nil, got %v", p.Email)
	}
	if p.FirstNameTH != "ทดสอบ" {
		t.Errorf("expected first_name_th 'ทดสอบ', got %q", p.FirstNameTH)
	}
}

func TestHospitalAService_SearchPatient_NotFound(t *testing.T) {
	upstream := newUpstreamServer(http.StatusNotFound, nil)
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	_, err := svc.SearchPatient(context.Background(), "unknown")
	if !errors.Is(err, service.ErrHospitalANotFound) {
		t.Errorf("expected ErrHospitalANotFound, got %v", err)
	}
}

func TestHospitalAService_SearchPatient_UpstreamServerError(t *testing.T) {
	upstream := newUpstreamServer(http.StatusInternalServerError, nil)
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	_, err := svc.SearchPatient(context.Background(), "1234567890123")
	if !errors.Is(err, service.ErrHospitalAUpstream) {
		t.Errorf("expected ErrHospitalAUpstream, got %v", err)
	}
}

func TestHospitalAService_SearchPatient_UpstreamBadGateway(t *testing.T) {
	upstream := newUpstreamServer(http.StatusBadGateway, nil)
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	_, err := svc.SearchPatient(context.Background(), "1234567890123")
	if !errors.Is(err, service.ErrHospitalAUpstream) {
		t.Errorf("expected ErrHospitalAUpstream, got %v", err)
	}
}

func TestHospitalAService_SearchPatient_ConnectionRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := srv.URL
	srv.Close()

	svc := service.NewHospitalAService(closedURL)
	_, err := svc.SearchPatient(context.Background(), "1234567890123")
	if !errors.Is(err, service.ErrHospitalAUpstream) {
		t.Errorf("expected ErrHospitalAUpstream, got %v", err)
	}
}

func TestHospitalAService_SearchPatient_InvalidJSON(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json")) //nolint:errcheck
	}))
	defer upstream.Close()

	svc := service.NewHospitalAService(upstream.URL)
	_, err := svc.SearchPatient(context.Background(), "1234567890123")
	if !errors.Is(err, service.ErrHospitalAUpstream) {
		t.Errorf("expected ErrHospitalAUpstream, got %v", err)
	}
}
