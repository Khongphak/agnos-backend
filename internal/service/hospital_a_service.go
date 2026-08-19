package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

var (
	ErrHospitalANotFound = errors.New("patient not found in hospital a")
	ErrHospitalAUpstream = errors.New("hospital a upstream error")
)

// hospitalAPatientDTO mirrors the JSON shape returned by Hospital A upstream API.
type hospitalAPatientDTO struct {
	NationalID  string `json:"national_id"`
	PassportID  string `json:"passport_id"`
	FirstNameTH string `json:"first_name_th"`
	LastNameTH  string `json:"last_name_th"`
	FirstNameEN string `json:"first_name_en"`
	LastNameEN  string `json:"last_name_en"`
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
}

// HospitalAService defines the contract for Hospital A proxy operations.
type HospitalAService interface {
	SearchPatient(ctx context.Context, id string) (*response.HospitalAPatientResponse, error)
}

type hospitalAService struct {
	client  *http.Client
	baseURL string
}

// NewHospitalAService returns a HospitalAService with a dedicated 5-second timeout HTTP client.
func NewHospitalAService(baseURL string) HospitalAService {
	return &hospitalAService{
		client:  &http.Client{Timeout: 5 * time.Second},
		baseURL: baseURL,
	}
}

func (s *hospitalAService) SearchPatient(ctx context.Context, id string) (*response.HospitalAPatientResponse, error) {
	endpoint := fmt.Sprintf("%s/patient/search/%s", s.baseURL, url.PathEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		log.Printf("hospital_a: failed to build request: %v", err)
		return nil, ErrHospitalAUpstream
	}

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("hospital_a: upstream request failed: %v", err)
		return nil, ErrHospitalAUpstream
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var dto hospitalAPatientDTO
		if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
			log.Printf("hospital_a: failed to decode response: %v", err)
			return nil, ErrHospitalAUpstream
		}
		return mapHospitalADTOToResponse(&dto), nil
	case http.StatusNotFound:
		return nil, ErrHospitalANotFound
	default:
		log.Printf("hospital_a: unexpected upstream status %d", resp.StatusCode)
		return nil, ErrHospitalAUpstream
	}
}

// mapHospitalADTOToResponse converts an upstream DTO to our canonical response shape.
// Fields absent from the upstream (empty string) become nil pointers per schema contract.
func mapHospitalADTOToResponse(dto *hospitalAPatientDTO) *response.HospitalAPatientResponse {
	r := &response.HospitalAPatientResponse{
		FirstNameTH: dto.FirstNameTH,
		LastNameTH:  dto.LastNameTH,
		DateOfBirth: dto.DateOfBirth,
		Gender:      dto.Gender,
	}
	if dto.NationalID != "" {
		v := dto.NationalID
		r.NationalID = &v
	}
	if dto.PassportID != "" {
		v := dto.PassportID
		r.PassportID = &v
	}
	if dto.FirstNameEN != "" {
		v := dto.FirstNameEN
		r.FirstNameEN = &v
	}
	if dto.LastNameEN != "" {
		v := dto.LastNameEN
		r.LastNameEN = &v
	}
	if dto.PhoneNumber != "" {
		v := dto.PhoneNumber
		r.PhoneNumber = &v
	}
	if dto.Email != "" {
		v := dto.Email
		r.Email = &v
	}
	return r
}
