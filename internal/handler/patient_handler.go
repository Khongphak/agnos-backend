package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/middleware"
	"github.com/agnos-assessment/agnos-backend/internal/model"
	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

// PatientHandler handles patient-related HTTP endpoints.
type PatientHandler struct {
	svc service.PatientService
}

func NewPatientHandler(svc service.PatientService) *PatientHandler {
	return &PatientHandler{svc: svc}
}

// Search godoc
// @Summary      Search patients
// @Description  Returns patients registered at the authenticated staff's hospital. All query params are optional; omitting all returns every patient for that hospital.
// @Tags         patient
// @Produce      json
// @Security     BearerAuth
// @Param        national_id  query  string  false  "Thai national ID (exact match)"
// @Param        passport_id  query  string  false  "Passport ID (exact match)"
// @Param        first_name   query  string  false  "First name partial match (ILIKE)"
// @Param        last_name    query  string  false  "Last name partial match (ILIKE)"
// @Param        dob          query  string  false  "Date of birth exact match (YYYY-MM-DD)"
// @Param        phone        query  string  false  "Phone number partial match (ILIKE)"
// @Param        email        query  string  false  "Email partial match (ILIKE)"
// @Success      200  {object}  response.PatientListResponse
// @Failure      401  {object}  response.ErrorResponse
// @Failure      500  {object}  response.ErrorResponse
// @Router       /patient/search [get]
func (h *PatientHandler) Search(c *gin.Context) {
	claims := c.MustGet(middleware.StaffClaimsKey).(*service.StaffClaims)

	dob := c.Query("dob")
	if dob != "" {
		if _, err := time.Parse("2006-01-02", dob); err != nil {
			c.JSON(http.StatusBadRequest, response.NewError("INVALID_INPUT", "dob must be in YYYY-MM-DD format"))
			return
		}
	}

	params := service.SearchParams{
		HospitalID: claims.HospitalID,
		NationalID: c.Query("national_id"),
		PassportID: c.Query("passport_id"),
		FirstName:  c.Query("first_name"),
		LastName:   c.Query("last_name"),
		DOB:        dob,
		Phone:      c.Query("phone"),
		Email:      c.Query("email"),
	}

	patients, err := h.svc.Search(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.NewError("INTERNAL_ERROR", "internal server error"))
		return
	}

	c.JSON(http.StatusOK, response.PatientListResponse{
		Patients: toPatientResults(patients),
	})
}

func toPatientResults(patients []model.Patient) []response.PatientResult {
	out := make([]response.PatientResult, len(patients))
	for i, p := range patients {
		out[i] = response.PatientResult{
			ID:          p.ID,
			NationalID:  p.NationalID,
			PassportID:  p.PassportID,
			FirstNameTH: p.FirstNameTH,
			LastNameTH:  p.LastNameTH,
			FirstNameEN: p.FirstNameEN,
			LastNameEN:  p.LastNameEN,
			DateOfBirth: p.DateOfBirth.Format("2006-01-02"),
			Gender:      p.Gender,
			PhoneNumber: p.PhoneNumber,
			Email:       p.Email,
			PatientHN:   p.PatientHN,
		}
	}
	return out
}
