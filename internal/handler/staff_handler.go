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

type createStaffRequest struct {
	Username     string `json:"username"      binding:"required"`
	Password     string `json:"password"      binding:"required"`
	HospitalCode string `json:"hospital_code" binding:"required"`
	Role         string `json:"role"`
}

type loginRequest struct {
	Username     string `json:"username"      binding:"required"`
	Password     string `json:"password"      binding:"required"`
	HospitalCode string `json:"hospital_code" binding:"required"`
}

func (h *StaffHandler) Create(c *gin.Context) {
	var req createStaffRequest
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

	c.JSON(http.StatusCreated, gin.H{
		"id":            result.ID,
		"username":      result.Username,
		"hospital_code": result.HospitalCode,
		"role":          result.Role,
	})
}

func (h *StaffHandler) Login(c *gin.Context) {
	var req loginRequest
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

	c.JSON(http.StatusOK, gin.H{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}
