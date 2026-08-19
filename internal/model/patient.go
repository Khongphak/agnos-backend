package model

import "time"

// Patient represents a patient row joined with patient_hospital_registrations.
type Patient struct {
	ID          int64
	NationalID  *string
	PassportID  *string
	FirstNameTH string
	LastNameTH  string
	FirstNameEN *string
	LastNameEN  *string
	DateOfBirth time.Time
	Gender      string
	PhoneNumber *string
	Email       *string
	PatientHN   string // from patient_hospital_registrations
}
