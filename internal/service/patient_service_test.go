package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
	"github.com/agnos-assessment/agnos-backend/internal/service"
)

// ── mock repository ───────────────────────────────────────────────────────────

type mockPatientRepo struct {
	hospital       *model.Hospital
	hospitalErr    error
	searchResult   []model.Patient
	searchErr      error
	capturedParams repository.SearchPatientParams
}

func (m *mockPatientRepo) FindHospitalByCode(_ context.Context, _ string) (*model.Hospital, error) {
	return m.hospital, m.hospitalErr
}

func (m *mockPatientRepo) SearchPatients(_ context.Context, params repository.SearchPatientParams) ([]model.Patient, error) {
	m.capturedParams = params
	return m.searchResult, m.searchErr
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestPatientService_Search_ForwardsParams(t *testing.T) {
	repo := &mockPatientRepo{
		searchResult: []model.Patient{{ID: 1, FirstNameTH: "สมชาย", LastNameTH: "ใจดี", PatientHN: "HN001"}},
	}
	svc := service.NewPatientService(repo)

	params := service.SearchParams{
		HospitalID: 7,
		NationalID: "1234567890123",
		FirstName:  "สม",
	}
	patients, err := svc.Search(context.Background(), params)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(patients) != 1 {
		t.Fatalf("expected 1 patient, got %d", len(patients))
	}
	if repo.capturedParams.HospitalID != 7 {
		t.Errorf("expected hospital_id=7 forwarded to repo, got %d", repo.capturedParams.HospitalID)
	}
	if repo.capturedParams.NationalID != "1234567890123" {
		t.Errorf("expected national_id forwarded, got %q", repo.capturedParams.NationalID)
	}
	if repo.capturedParams.FirstName != "สม" {
		t.Errorf("expected first_name forwarded, got %q", repo.capturedParams.FirstName)
	}
}

func TestPatientService_Search_EmptyResult(t *testing.T) {
	repo := &mockPatientRepo{searchResult: nil}
	svc := service.NewPatientService(repo)

	patients, err := svc.Search(context.Background(), service.SearchParams{HospitalID: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if patients != nil {
		t.Errorf("expected nil/empty slice, got %v", patients)
	}
}

func TestPatientService_Search_RepoError(t *testing.T) {
	repo := &mockPatientRepo{searchErr: errors.New("db error")}
	svc := service.NewPatientService(repo)

	_, err := svc.Search(context.Background(), service.SearchParams{HospitalID: 1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPatientService_FindHospitalByCode_Found(t *testing.T) {
	repo := &mockPatientRepo{hospital: &model.Hospital{ID: 3, Code: "BKK001", Name: "Bangkok Hospital"}}
	svc := service.NewPatientService(repo)

	h, err := svc.FindHospitalByCode(context.Background(), "BKK001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.ID != 3 {
		t.Errorf("expected hospital ID 3, got %d", h.ID)
	}
}

func TestPatientService_FindHospitalByCode_NotFound(t *testing.T) {
	repo := &mockPatientRepo{hospitalErr: repository.ErrNotFound}
	svc := service.NewPatientService(repo)

	_, err := svc.FindHospitalByCode(context.Background(), "UNKNOWN")
	if !errors.Is(err, service.ErrPatientHospitalNotFound) {
		t.Errorf("expected ErrPatientHospitalNotFound, got %v", err)
	}
}
