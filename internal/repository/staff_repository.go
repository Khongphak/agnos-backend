package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"

	"github.com/agnos-assessment/agnos-backend/internal/model"
)

var (
	ErrNotFound          = errors.New("record not found")
	ErrDuplicateUsername = errors.New("username already exists in this hospital")
)

type StaffRepository interface {
	FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error)
	CreateStaff(ctx context.Context, hospitalID int64, username, passwordHash, role string) (*model.Staff, error)
	FindStaffByUsernameAndHospital(ctx context.Context, username string, hospitalID int64) (*model.Staff, error)
	SaveRefreshTokenAndUpdateLoginTime(ctx context.Context, staffID int64, tokenHash string, expiresAt time.Time) error
	FindStaffByID(ctx context.Context, staffID int64) (*model.Staff, error)
	FindRefreshToken(ctx context.Context, tokenHash string) (*model.StaffRefreshToken, error)
	DeleteRefreshToken(ctx context.Context, tokenHash string) error
	RotateRefreshToken(ctx context.Context, staffID int64, oldHash, newHash string, newExpiresAt time.Time) error
}

type staffRepository struct {
	db *sql.DB
}

var _ StaffRepository = (*staffRepository)(nil)

func NewStaffRepository(db *sql.DB) StaffRepository {
	return &staffRepository{db: db}
}

func (r *staffRepository) FindHospitalByCode(ctx context.Context, code string) (*model.Hospital, error) {
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

func (r *staffRepository) CreateStaff(ctx context.Context, hospitalID int64, username, passwordHash, role string) (*model.Staff, error) {
	var s model.Staff
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO staff (hospital_id, username, password_hash, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, hospital_id, username, role, is_active, created_at, updated_at`,
		hospitalID, username, passwordHash, role,
	).Scan(&s.ID, &s.HospitalID, &s.Username, &s.Role, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, ErrDuplicateUsername
		}
		return nil, err
	}
	return &s, nil
}

func (r *staffRepository) FindStaffByUsernameAndHospital(ctx context.Context, username string, hospitalID int64) (*model.Staff, error) {
	var s model.Staff
	var lastLoginAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, hospital_id, username, password_hash, role, is_active, last_login_at, created_at, updated_at
		 FROM staff WHERE username = $1 AND hospital_id = $2`,
		username, hospitalID,
	).Scan(&s.ID, &s.HospitalID, &s.Username, &s.PasswordHash, &s.Role, &s.IsActive, &lastLoginAt, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastLoginAt.Valid {
		s.LastLoginAt = &lastLoginAt.Time
	}
	return &s, nil
}

func (r *staffRepository) SaveRefreshTokenAndUpdateLoginTime(ctx context.Context, staffID int64, tokenHash string, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err = tx.ExecContext(ctx,
		`INSERT INTO staff_refresh_tokens (staff_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		staffID, tokenHash, expiresAt,
	); err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE staff SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`,
		staffID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *staffRepository) FindStaffByID(ctx context.Context, staffID int64) (*model.Staff, error) {
	var s model.Staff
	err := r.db.QueryRowContext(ctx,
		`SELECT id, hospital_id, username, role, is_active FROM staff WHERE id = $1`,
		staffID,
	).Scan(&s.ID, &s.HospitalID, &s.Username, &s.Role, &s.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *staffRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*model.StaffRefreshToken, error) {
	var t model.StaffRefreshToken
	err := r.db.QueryRowContext(ctx,
		`SELECT id, staff_id, token_hash, expires_at, created_at
		 FROM staff_refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	).Scan(&t.ID, &t.StaffID, &t.TokenHash, &t.ExpiresAt, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *staffRepository) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM staff_refresh_tokens WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *staffRepository) RotateRefreshToken(ctx context.Context, staffID int64, oldHash, newHash string, newExpiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	res, err := tx.ExecContext(ctx,
		`DELETE FROM staff_refresh_tokens WHERE token_hash = $1`, oldHash)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}

	if _, err = tx.ExecContext(ctx,
		`INSERT INTO staff_refresh_tokens (staff_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		staffID, newHash, newExpiresAt,
	); err != nil {
		return err
	}

	return tx.Commit()
}
