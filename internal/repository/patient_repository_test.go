package repository_test

import (
	"strings"
	"testing"

	"github.com/agnos-assessment/agnos-backend/internal/repository"
)

// ── BuildSearchQuery unit tests ───────────────────────────────────────────────

func TestBuildSearchQuery_NoFilters(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{HospitalID: 5})
	if len(args) != 1 {
		t.Fatalf("expected 1 arg (hospital_id), got %d", len(args))
	}
	if args[0] != int64(5) {
		t.Errorf("expected hospital_id=5, got %v", args[0])
	}
	if !strings.Contains(q, "phr.hospital_id = $1") {
		t.Error("expected $1 placeholder for hospital_id in query")
	}
	// no extra AND clauses
	if strings.Contains(q, "$2") {
		t.Error("unexpected $2 in query when no filters are set")
	}
}

func TestBuildSearchQuery_NationalID(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		NationalID: "1234567890123",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "1234567890123" {
		t.Errorf("expected national_id arg, got %v", args[1])
	}
	if !strings.Contains(q, "p.national_id = $2") {
		t.Error("expected national_id condition in query")
	}
}

func TestBuildSearchQuery_PassportID(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		PassportID: "AB123456",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "AB123456" {
		t.Errorf("expected passport_id arg, got %v", args[1])
	}
	if !strings.Contains(q, "p.passport_id = $2") {
		t.Error("expected passport_id condition in query")
	}
}

func TestBuildSearchQuery_FirstName(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		FirstName:  "John",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args (hospital_id + first_name), got %d", len(args))
	}
	if args[1] != "%John%" {
		t.Errorf("expected '%%John%%' for ILIKE arg, got %v", args[1])
	}
	// same $2 referenced twice for both columns
	if !strings.Contains(q, "p.first_name_th ILIKE $2") {
		t.Error("expected first_name_th ILIKE $2")
	}
	if !strings.Contains(q, "p.first_name_en ILIKE $2") {
		t.Error("expected first_name_en ILIKE $2")
	}
}

func TestBuildSearchQuery_LastName(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		LastName:   "Doe",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "%Doe%" {
		t.Errorf("expected '%%Doe%%', got %v", args[1])
	}
	if !strings.Contains(q, "p.last_name_th ILIKE $2") {
		t.Error("expected last_name_th ILIKE $2")
	}
}

func TestBuildSearchQuery_DOB(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		DOB:        "1990-01-15",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "1990-01-15" {
		t.Errorf("expected DOB arg '1990-01-15', got %v", args[1])
	}
	if !strings.Contains(q, "p.date_of_birth = $2") {
		t.Error("expected date_of_birth condition in query")
	}
}

func TestBuildSearchQuery_Phone(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		Phone:      "081",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "%081%" {
		t.Errorf("expected '%%081%%', got %v", args[1])
	}
	if !strings.Contains(q, "p.phone_number ILIKE $2") {
		t.Error("expected phone_number condition in query")
	}
}

func TestBuildSearchQuery_Email(t *testing.T) {
	q, args := repository.BuildSearchQuery(repository.SearchPatientParams{
		HospitalID: 1,
		Email:      "john@",
	})
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[1] != "%john@%" {
		t.Errorf("expected '%%john@%%', got %v", args[1])
	}
	if !strings.Contains(q, "p.email ILIKE $2") {
		t.Error("expected email condition in query")
	}
}

func TestBuildSearchQuery_AllFilters(t *testing.T) {
	params := repository.SearchPatientParams{
		HospitalID: 42,
		NationalID: "1234567890123",
		PassportID: "AB123456",
		FirstName:  "John",
		LastName:   "Doe",
		DOB:        "1990-01-15",
		Phone:      "081",
		Email:      "john@",
	}
	_, args := repository.BuildSearchQuery(params)
	// hospital_id + 7 filter args = 8 total
	if len(args) != 8 {
		t.Fatalf("expected 8 args for all filters, got %d", len(args))
	}
}

func TestBuildSearchQuery_MultipleFilters_CorrectIndexing(t *testing.T) {
	// national_id=$2, first_name=$3, email=$4
	params := repository.SearchPatientParams{
		HospitalID: 1,
		NationalID: "111",
		FirstName:  "Jo",
		Email:      "jo@",
	}
	q, args := repository.BuildSearchQuery(params)
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
	if !strings.Contains(q, "p.national_id = $2") {
		t.Error("expected national_id = $2")
	}
	if !strings.Contains(q, "ILIKE $3") {
		t.Error("expected ILIKE $3 for first_name")
	}
	if !strings.Contains(q, "p.email ILIKE $4") {
		t.Error("expected email ILIKE $4")
	}
}
