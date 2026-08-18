package model

import "time"

type Hospital struct {
	ID   int64
	Name string
	Code string
}

type Staff struct {
	ID           int64
	HospitalID   int64
	Username     string
	PasswordHash string
	Role         string
	IsActive     bool
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
