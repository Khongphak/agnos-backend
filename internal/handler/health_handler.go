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

// GetHealth godoc
// @Summary      Health check
// @Description  Returns server and database status
// @Tags         health
// @Produce      json
// @Success      200  {object}  response.HealthResponse
// @Failure      503  {object}  response.ErrorResponse
// @Router       /health [get]
func (h *HealthHandler) GetHealth(c *gin.Context) {
	if err := h.svc.Check(c.Request.Context()); err != nil {
		c.JSON(http.StatusServiceUnavailable, response.NewError("DB_UNAVAILABLE", "database unavailable"))
		return
	}
	c.JSON(http.StatusOK, response.HealthResponse{Status: "ok", Database: "connected"})
}
