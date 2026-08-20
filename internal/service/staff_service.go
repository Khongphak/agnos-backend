package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/repository"
)

const (
	bcryptCost      = 12
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour
	rawTokenBytes   = 32
)

var (
	ErrHospitalNotFound   = errors.New("hospital not found")
	ErrUsernameConflict   = errors.New("username already taken in this hospital")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrTokenNotFound      = errors.New("refresh token not found or already revoked")
	ErrTokenExpired       = errors.New("refresh token expired")
)

// StaffClaims is the JWT payload for staff access tokens.
// Exported so auth middleware can parse tokens into this type.
type StaffClaims struct {
	StaffID    int64  `json:"staff_id"`
	HospitalID int64  `json:"hospital_id"`
	Role       string `json:"role"`
	jwt.RegisteredClaims
}

type CreateStaffRequest struct {
	Username     string
	Password     string
	HospitalCode string
	Role         string
}

type CreateStaffResponse struct {
	ID           int64
	Username     string
	HospitalCode string
	Role         string
}

type LoginRequest struct {
	Username     string
	Password     string
	HospitalCode string
}

type LoginResponse struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
}

type StaffService interface {
	Create(ctx context.Context, req CreateStaffRequest) (*CreateStaffResponse, error)
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
	Refresh(ctx context.Context, rawToken string) (*LoginResponse, error)
	Logout(ctx context.Context, rawToken string) error
}

type staffService struct {
	repo      repository.StaffRepository
	jwtSecret []byte
}

func NewStaffService(repo repository.StaffRepository, jwtSecret string) StaffService {
	return &staffService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

func (s *staffService) Create(ctx context.Context, req CreateStaffRequest) (*CreateStaffResponse, error) {
	hospital, err := s.repo.FindHospitalByCode(ctx, req.HospitalCode)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrHospitalNotFound
	}
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		return nil, err
	}

	staff, err := s.repo.CreateStaff(ctx, hospital.ID, req.Username, string(hash), req.Role)
	if errors.Is(err, repository.ErrDuplicateUsername) {
		return nil, ErrUsernameConflict
	}
	if err != nil {
		return nil, err
	}

	return &CreateStaffResponse{
		ID:           staff.ID,
		Username:     staff.Username,
		HospitalCode: req.HospitalCode,
		Role:         staff.Role,
	}, nil
}

func (s *staffService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	hospital, err := s.repo.FindHospitalByCode(ctx, req.HospitalCode)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	staff, err := s.repo.FindStaffByUsernameAndHospital(ctx, req.Username, hospital.ID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(staff.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if !staff.IsActive {
		return nil, ErrAccountInactive
	}

	accessToken, err := s.generateAccessToken(staff)
	if err != nil {
		return nil, err
	}

	rawToken, tokenHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(refreshTokenTTL)
	if err := s.repo.SaveRefreshTokenAndUpdateLoginTime(ctx, staff.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawToken,
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

func (s *staffService) generateAccessToken(staff *model.Staff) (string, error) {
	now := time.Now()
	claims := StaffClaims{
		StaffID:    staff.ID,
		HospitalID: staff.HospitalID,
		Role:       staff.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func generateRefreshToken() (raw, hash string, err error) {
	b := make([]byte, rawTokenBytes)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	sum := sha256.Sum256(b)
	hash = hex.EncodeToString(sum[:])
	return raw, hash, nil
}

// hashRefreshToken re-derives the stored SHA-256 hash from a raw token string.
// The raw token is hex-encoded bytes, so we decode → hash the original bytes.
func hashRefreshToken(raw string) (string, error) {
	b, err := hex.DecodeString(raw)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (s *staffService) Refresh(ctx context.Context, rawToken string) (*LoginResponse, error) {
	tokenHash, err := hashRefreshToken(rawToken)
	if err != nil {
		return nil, ErrTokenNotFound
	}

	rt, err := s.repo.FindRefreshToken(ctx, tokenHash)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(rt.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	staff, err := s.repo.FindStaffByID(ctx, rt.StaffID)
	if err != nil {
		return nil, err
	}

	accessToken, err := s.generateAccessToken(staff)
	if err != nil {
		return nil, err
	}

	newRaw, newHash, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	if err := s.repo.RotateRefreshToken(ctx, staff.ID, tokenHash, newHash, time.Now().Add(refreshTokenTTL)); err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		ExpiresIn:    int64(accessTokenTTL.Seconds()),
	}, nil
}

func (s *staffService) Logout(ctx context.Context, rawToken string) error {
	tokenHash, err := hashRefreshToken(rawToken)
	if err != nil {
		return ErrTokenNotFound
	}

	if err := s.repo.DeleteRefreshToken(ctx, tokenHash); errors.Is(err, repository.ErrNotFound) {
		return ErrTokenNotFound
	} else if err != nil {
		return err
	}
	return nil
}
