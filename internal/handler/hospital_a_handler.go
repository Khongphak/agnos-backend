package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

// HospitalAHandler proxies patient lookups to the Hospital A upstream API.
type HospitalAHandler struct {
	svc service.HospitalAService
}

func NewHospitalAHandler(svc service.HospitalAService) *HospitalAHandler {
	return &HospitalAHandler{svc: svc}
}

// SearchPatient godoc
//
//	@Summary		Search patient by ID via Hospital A
//	@Description	Proxies a patient lookup to Hospital A upstream API using a Thai national ID or passport ID. No auth required. Fields absent from the upstream response are null.
//	@Tags			hospital-a
//	@Produce		json
//	@Param			id	path		string	true	"Thai national ID or passport ID"
//	@Success		200	{object}	response.HospitalAPatientResponse
//	@Failure		400	{object}	response.ErrorResponse	"INVALID_REQUEST — id is empty"
//	@Failure		404	{object}	response.ErrorResponse	"NOT_FOUND — patient not found in Hospital A"
//	@Failure		502	{object}	response.ErrorResponse	"UPSTREAM_ERROR — timeout, connection error, or unexpected upstream status"
//	@Router			/hospital-a/patient/search/{id} [get]
func (h *HospitalAHandler) SearchPatient(c *gin.Context) {
	id := c.Param("id")
	if strings.TrimSpace(id) == "" {
		c.JSON(http.StatusBadRequest, response.NewError("INVALID_REQUEST", "id is required"))
		return
	}

	patient, err := h.svc.SearchPatient(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrHospitalANotFound):
			c.JSON(http.StatusNotFound, response.NewError("NOT_FOUND", "patient not found"))
		default:
			c.JSON(http.StatusBadGateway, response.NewError("UPSTREAM_ERROR", "upstream request failed"))
		}
		return
	}

	c.JSON(http.StatusOK, patient)
}
