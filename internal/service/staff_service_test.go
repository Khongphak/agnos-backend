package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
	"github.com/agnos-assessment/agnos-backend/internal/service"
)

const testJWTSecret = "test-secret-key"

// mockStaffRepo implements repository.StaffRepository for testing.
type mockStaffRepo struct {
	hospital        *model.Hospital
	findHospitalErr error

	createdStaff  *model.Staff
	createStaffErr error

	foundStaff    *model.Staff
	findStaffErr  error

	saveTokenErr error
}

func (m *mockStaffRepo) FindHospitalByCode(_ context.Context, _ string) (*model.Hospital, error) {
	return m.hospital, m.findHospitalErr
}

func (m *mockStaffRepo) CreateStaff(_ context.Context, _ int64, _, _, _ string) (*model.Staff, error) {
	return m.createdStaff, m.createStaffErr
}

func (m *mockStaffRepo) FindStaffByUsernameAndHospital(_ context.Context, _ string, _ int64) (*model.Staff, error) {
	return m.foundStaff, m.findStaffErr
}

func (m *mockStaffRepo) SaveRefreshTokenAndUpdateLoginTime(_ context.Context, _ int64, _ string, _ time.Time) error {
	return m.saveTokenErr
}

func newTestStaffService(repo repository.StaffRepository) service.StaffService {
	return service.NewStaffService(repo, testJWTSecret)
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestStaffService_Create_Success(t *testing.T) {
	repo := &mockStaffRepo{
		hospital: &model.Hospital{ID: 1, Code: "HOSP01"},
		createdStaff: &model.Staff{
			ID:         42,
			HospitalID: 1,
			Username:   "alice",
			Role:       "staff",
			IsActive:   true,
		},
	}
	svc := newTestStaffService(repo)

	resp, err := svc.Create(context.Background(), service.CreateStaffRequest{
		Username:     "alice",
		Password:     "SecurePass123",
		HospitalCode: "HOSP01",
		Role:         "staff",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.ID != 42 {
		t.Errorf("expected ID 42, got %d", resp.ID)
	}
	if resp.Username != "alice" {
		t.Errorf("expected username 'alice', got %q", resp.Username)
	}
	if resp.HospitalCode != "HOSP01" {
		t.Errorf("expected hospital_code 'HOSP01', got %q", resp.HospitalCode)
	}
	if resp.Role != "staff" {
		t.Errorf("expected role 'staff', got %q", resp.Role)
	}
}

func TestStaffService_Create_HospitalNotFound(t *testing.T) {
	repo := &mockStaffRepo{findHospitalErr: repository.ErrNotFound}
	svc := newTestStaffService(repo)

	_, err := svc.Create(context.Background(), service.CreateStaffRequest{
		Username:     "alice",
		Password:     "pass",
		HospitalCode: "UNKNOWN",
		Role:         "staff",
	})
	if !errors.Is(err, service.ErrHospitalNotFound) {
		t.Errorf("expected ErrHospitalNotFound, got %v", err)
	}
}

func TestStaffService_Create_UsernameConflict(t *testing.T) {
	repo := &mockStaffRepo{
		hospital:       &model.Hospital{ID: 1, Code: "HOSP01"},
		createStaffErr: repository.ErrDuplicateUsername,
	}
	svc := newTestStaffService(repo)

	_, err := svc.Create(context.Background(), service.CreateStaffRequest{
		Username:     "alice",
		Password:     "pass",
		HospitalCode: "HOSP01",
		Role:         "staff",
	})
	if !errors.Is(err, service.ErrUsernameConflict) {
		t.Errorf("expected ErrUsernameConflict, got %v", err)
	}
}

func TestStaffService_Create_RepoError(t *testing.T) {
	dbErr := errors.New("connection reset")
	repo := &mockStaffRepo{findHospitalErr: dbErr}
	svc := newTestStaffService(repo)

	_, err := svc.Create(context.Background(), service.CreateStaffRequest{
		Username: "alice", Password: "pass", HospitalCode: "H1", Role: "staff",
	})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected wrapped db error, got %v", err)
	}
}

// ── Login ────────────────────────────────────────────────────────────────────

// precomputedHash is a bcrypt hash of "correctpassword" at cost 4 (fast for tests).
var precomputedHash string

func init() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), 4)
	precomputedHash = string(hash)
}

func activeStaff() *model.Staff {
	return &model.Staff{
		ID:           7,
		HospitalID:   1,
		Username:     "bob",
		PasswordHash: precomputedHash,
		Role:         "admin",
		IsActive:     true,
	}
}

func TestStaffService_Login_Success(t *testing.T) {
	repo := &mockStaffRepo{
		hospital:   &model.Hospital{ID: 1, Code: "HOSP01"},
		foundStaff: activeStaff(),
	}
	svc := newTestStaffService(repo)

	resp, err := svc.Login(context.Background(), service.LoginRequest{
		Username:     "bob",
		Password:     "correctpassword",
		HospitalCode: "HOSP01",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("access_token must not be empty")
	}
	if resp.RefreshToken == "" {
		t.Error("refresh_token must not be empty")
	}
	if resp.ExpiresIn != 900 {
		t.Errorf("expected expires_in 900, got %d", resp.ExpiresIn)
	}

	// Verify JWT claims.
	claims := &service.StaffClaims{}
	token, err := jwt.ParseWithClaims(resp.AccessToken, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	if err != nil || !token.Valid {
		t.Fatalf("access token invalid: %v", err)
	}
	if claims.StaffID != 7 {
		t.Errorf("expected staff_id 7, got %d", claims.StaffID)
	}
	if claims.HospitalID != 1 {
		t.Errorf("expected hospital_id 1, got %d", claims.HospitalID)
	}
	if claims.Role != "admin" {
		t.Errorf("expected role 'admin', got %q", claims.Role)
	}
}

func TestStaffService_Login_HospitalNotFound(t *testing.T) {
	repo := &mockStaffRepo{findHospitalErr: repository.ErrNotFound}
	svc := newTestStaffService(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "bob", Password: "pass", HospitalCode: "BAD",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestStaffService_Login_StaffNotFound(t *testing.T) {
	repo := &mockStaffRepo{
		hospital:     &model.Hospital{ID: 1, Code: "HOSP01"},
		findStaffErr: repository.ErrNotFound,
	}
	svc := newTestStaffService(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "nobody", Password: "pass", HospitalCode: "HOSP01",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestStaffService_Login_WrongPassword(t *testing.T) {
	repo := &mockStaffRepo{
		hospital:   &model.Hospital{ID: 1, Code: "HOSP01"},
		foundStaff: activeStaff(),
	}
	svc := newTestStaffService(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "bob", Password: "wrongpassword", HospitalCode: "HOSP01",
	})
	if !errors.Is(err, service.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestStaffService_Login_AccountInactive(t *testing.T) {
	inactive := activeStaff()
	inactive.IsActive = false
	repo := &mockStaffRepo{
		hospital:   &model.Hospital{ID: 1, Code: "HOSP01"},
		foundStaff: inactive,
	}
	svc := newTestStaffService(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "bob", Password: "correctpassword", HospitalCode: "HOSP01",
	})
	if !errors.Is(err, service.ErrAccountInactive) {
		t.Errorf("expected ErrAccountInactive, got %v", err)
	}
}

func TestStaffService_Login_SaveTokenError(t *testing.T) {
	dbErr := errors.New("tx failed")
	repo := &mockStaffRepo{
		hospital:     &model.Hospital{ID: 1, Code: "HOSP01"},
		foundStaff:   activeStaff(),
		saveTokenErr: dbErr,
	}
	svc := newTestStaffService(repo)

	_, err := svc.Login(context.Background(), service.LoginRequest{
		Username: "bob", Password: "correctpassword", HospitalCode: "HOSP01",
	})
	if !errors.Is(err, dbErr) {
		t.Errorf("expected db error, got %v", err)
	}
}
