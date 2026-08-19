package service

import (
	"context"
	"errors"

	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
)

var ErrPatientHospitalNotFound = errors.New("hospital not found")

// SearchParams mirrors repository.SearchPatientParams at the service boundary.
type SearchParams struct {
	HospitalID int64
	NationalID string
	PassportID string
	FirstName  string
	LastName   string
	DOB        string
	Phone      string
	Email      string
}

// PatientService defines business logic for patient operations.
type PatientService interface {
	Search(ctx context.Context, params SearchParams) ([]model.Patient, error)
	FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error)
}

type patientService struct {
	repo repository.PatientRepository
}

func NewPatientService(repo repository.PatientRepository) PatientService {
	return &patientService{repo: repo}
}

func (s *patientService) Search(ctx context.Context, params SearchParams) ([]model.Patient, error) {
	return s.repo.SearchPatients(ctx, repository.SearchPatientParams{
		HospitalID: params.HospitalID,
		NationalID: params.NationalID,
		PassportID: params.PassportID,
		FirstName:  params.FirstName,
		LastName:   params.LastName,
		DOB:        params.DOB,
		Phone:      params.Phone,
		Email:      params.Email,
	})
}

func (s *patientService) FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error) {
	h, err := s.repo.FindHospitalByCode(ctx, code)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrPatientHospitalNotFound
	}
	return h, err
}
