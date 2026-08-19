package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/agnos-assessment/agnos-backend/internal/model"
)

// SearchPatientParams holds optional filters for patient search scoped to one hospital.
type SearchPatientParams struct {
	HospitalID int64
	NationalID string
	PassportID string
	FirstName  string // ILIKE on first_name_th OR first_name_en
	LastName   string // ILIKE on last_name_th OR last_name_en
	DOB        string // exact DATE, format "YYYY-MM-DD"
	Phone      string // ILIKE
	Email      string // ILIKE
}

// PatientRepository defines data access for patients.
type PatientRepository interface {
	FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error)
	SearchPatients(ctx context.Context, params SearchPatientParams) ([]model.Patient, error)
}

type patientRepository struct {
	db *sql.DB
}

var _ PatientRepository = (*patientRepository)(nil)

func NewPatientRepository(db *sql.DB) PatientRepository {
	return &patientRepository{db: db}
}

func (r *patientRepository) FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error) {
	var h model.Hospital
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, code FROM hospitals WHERE code = $1`, code,
	).Scan(&h.ID, &h.Name, &h.Code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (r *patientRepository) SearchPatients(ctx context.Context, params SearchPatientParams) ([]model.Patient, error) {
	query, args := BuildSearchQuery(params)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patients []model.Patient
	for rows.Next() {
		var p model.Patient
		var natID, passID, fnEN, lnEN, phone, email sql.NullString
		if err := rows.Scan(
			&p.ID, &natID, &passID,
			&p.FirstNameTH, &p.LastNameTH,
			&fnEN, &lnEN,
			&p.DateOfBirth, &p.Gender,
			&phone, &email, &p.PatientHN,
		); err != nil {
			return nil, err
		}
		if natID.Valid {
			p.NationalID = &natID.String
		}
		if passID.Valid {
			p.PassportID = &passID.String
		}
		if fnEN.Valid {
			p.FirstNameEN = &fnEN.String
		}
		if lnEN.Valid {
			p.LastNameEN = &lnEN.String
		}
		if phone.Valid {
			p.PhoneNumber = &phone.String
		}
		if email.Valid {
			p.Email = &email.String
		}
		patients = append(patients, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return patients, nil
}

// BuildSearchQuery constructs the parameterised SQL query for patient search.
// Exported to allow direct unit testing of query building logic.
func BuildSearchQuery(params SearchPatientParams) (string, []interface{}) {
	args := []interface{}{params.HospitalID}
	n := 2 // next $N index

	var sb strings.Builder
	sb.WriteString(
		`SELECT p.id, p.national_id, p.passport_id,
		        p.first_name_th, p.last_name_th,
		        p.first_name_en, p.last_name_en,
		        p.date_of_birth, p.gender,
		        p.phone_number, p.email, phr.patient_hn
		 FROM patients p
		 INNER JOIN patient_hospital_registrations phr
		         ON phr.patient_id = p.id AND phr.hospital_id = $1
		 WHERE 1=1`)

	if params.NationalID != "" {
		sb.WriteString(fmt.Sprintf(" AND p.national_id = $%d", n))
		args = append(args, params.NationalID)
		n++
	}
	if params.PassportID != "" {
		sb.WriteString(fmt.Sprintf(" AND p.passport_id = $%d", n))
		args = append(args, params.PassportID)
		n++
	}
	if params.FirstName != "" {
		// reuse same $N for both columns — PostgreSQL allows positional reuse
		sb.WriteString(fmt.Sprintf(" AND (p.first_name_th ILIKE $%d OR p.first_name_en ILIKE $%d)", n, n))
		args = append(args, "%"+params.FirstName+"%")
		n++
	}
	if params.LastName != "" {
		sb.WriteString(fmt.Sprintf(" AND (p.last_name_th ILIKE $%d OR p.last_name_en ILIKE $%d)", n, n))
		args = append(args, "%"+params.LastName+"%")
		n++
	}
	if params.DOB != "" {
		sb.WriteString(fmt.Sprintf(" AND p.date_of_birth = $%d", n))
		args = append(args, params.DOB)
		n++
	}
	if params.Phone != "" {
		sb.WriteString(fmt.Sprintf(" AND p.phone_number ILIKE $%d", n))
		args = append(args, "%"+params.Phone+"%")
		n++
	}
	if params.Email != "" {
		sb.WriteString(fmt.Sprintf(" AND p.email ILIKE $%d", n))
		args = append(args, "%"+params.Email+"%")
		n++
	}

	_ = n // suppress unused warning when all params are empty
	return sb.String(), args
}
