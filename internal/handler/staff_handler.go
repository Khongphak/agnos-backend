package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type StaffHandler struct {
	svc service.StaffService
}

func NewStaffHandler(svc service.StaffService) *StaffHandler {
	return &StaffHandler{svc: svc}
}

// CreateStaffInput is the request body for POST /staff/create.
type CreateStaffInput struct {
	Username     string `json:"username"      binding:"required" example:"nurse01"`
	Password     string `json:"password"      binding:"required" example:"P@ssw0rd123"`
	HospitalCode string `json:"hospital_code" binding:"required" example:"BKK001"`
	Role         string `json:"role"                             example:"staff"    enums:"staff,admin"`
}

// LoginInput is the request body for POST /staff/login.
type LoginInput struct {
	Username     string `json:"username"      binding:"required" example:"nurse01"`
	Password     string `json:"password"      binding:"required" example:"P@ssw0rd123"`
	HospitalCode string `json:"hospital_code" binding:"required" example:"BKK001"`
}

// Create godoc
//
//	@Summary		Create staff account
//	@Description	Registers a new staff member linked to a hospital
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CreateStaffInput	true	"Staff registration data"
//	@Success		201		{object}	response.StaffCreatedResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		404		{object}	response.ErrorResponse
//	@Failure		409		{object}	response.ErrorResponse
//	@Router			/staff/create [post]
func (h *StaffHandler) Create(c *gin.Context) {
	var req CreateStaffInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", err.Error()))
		return
	}

	if req.Role == "" {
		req.Role = "staff"
	}
	if req.Role != "staff" && req.Role != "admin" {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", "role must be 'staff' or 'admin'"))
		return
	}

	result, err := h.svc.Create(c.Request.Context(), service.CreateStaffRequest{
		Username:     req.Username,
		Password:     req.Password,
		HospitalCode: req.HospitalCode,
		Role:         req.Role,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrHospitalNotFound):
			c.JSON(http.StatusNotFound, response.NewError("HOSPITAL_NOT_FOUND", "hospital not found"))
		case errors.Is(err, service.ErrUsernameConflict):
			c.JSON(http.StatusConflict, response.NewError("USERNAME_CONFLICT", "username already exists in this hospital"))
		default:
			c.JSON(http.StatusInternalServerError, response.NewError("INTERNAL_ERROR", "internal server error"))
		}
		return
	}

	c.JSON(http.StatusCreated, response.StaffCreatedResponse{
		ID:           result.ID,
		Username:     result.Username,
		HospitalCode: result.HospitalCode,
		Role:         result.Role,
	})
}

// Login godoc
//
//	@Summary		Staff login
//	@Description	Authenticates a staff member and returns JWT access + refresh tokens
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			body	body		LoginInput	true	"Login credentials"
//	@Success		200		{object}	response.TokenResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Failure		403		{object}	response.ErrorResponse
//	@Router			/staff/login [post]
func (h *StaffHandler) Login(c *gin.Context) {
	var req LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", err.Error()))
		return
	}

	result, err := h.svc.Login(c.Request.Context(), service.LoginRequest{
		Username:     req.Username,
		Password:     req.Password,
		HospitalCode: req.HospitalCode,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, response.NewError("INVALID_CREDENTIALS", "invalid username, password, or hospital code"))
		case errors.Is(err, service.ErrAccountInactive):
			c.JSON(http.StatusForbidden, response.NewError("ACCOUNT_INACTIVE", "account is inactive"))
		default:
			c.JSON(http.StatusInternalServerError, response.NewError("INTERNAL_ERROR", "internal server error"))
		}
		return
	}

	c.JSON(http.StatusOK, response.TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// RefreshInput is the request body for POST /staff/refresh.
type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a1b2c3d4..."`
}

// LogoutInput is the request body for POST /staff/logout.
type LogoutInput struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"a1b2c3d4..."`
}

// Refresh godoc
//
//	@Summary		Refresh access token
//	@Description	Validates refresh token, rotates it, and returns a new access + refresh token pair
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			body	body		RefreshInput	true	"Refresh token"
//	@Success		200		{object}	response.TokenResponse
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Router			/staff/refresh [post]
func (h *StaffHandler) Refresh(c *gin.Context) {
	var req RefreshInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", err.Error()))
		return
	}

	result, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTokenNotFound), errors.Is(err, service.ErrTokenExpired):
			c.JSON(http.StatusUnauthorized, response.NewError("TOKEN_INVALID", "invalid or expired refresh token"))
		default:
			c.JSON(http.StatusInternalServerError, response.NewError("INTERNAL_ERROR", "internal server error"))
		}
		return
	}

	c.JSON(http.StatusOK, response.TokenResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// Logout godoc
//
//	@Summary		Staff logout
//	@Description	Revokes the refresh token, ending the session
//	@Tags			staff
//	@Accept			json
//	@Produce		json
//	@Param			body	body	LogoutInput	true	"Refresh token to revoke"
//	@Success		204		"No Content"
//	@Failure		400		{object}	response.ErrorResponse
//	@Failure		401		{object}	response.ErrorResponse
//	@Router			/staff/logout [post]
func (h *StaffHandler) Logout(c *gin.Context) {
	var req LogoutInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", err.Error()))
		return
	}

	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		switch {
		case errors.Is(err, service.ErrTokenNotFound):
			c.JSON(http.StatusUnauthorized, response.NewError("TOKEN_INVALID", "invalid or expired refresh token"))
		default:
			c.JSON(http.StatusInternalServerError, response.NewError("INTERNAL_ERROR", "internal server error"))
		}
		return
	}

	c.Status(http.StatusNoContent)
}
