package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/agnos-assessment/agnos-backend/internal/service"
	"github.com/agnos-assessment/agnos-backend/pkg/response"
)

type HealthHandler struct {
	svc service.HealthService
}

func NewHealthHandler(svc service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) GetHealth(c *gin.Context) {
	status, db := h.svc.Check()
	c.JSON(http.StatusOK, response.HealthResponse{
		Status:   status,
		Database: db,
	})
}
