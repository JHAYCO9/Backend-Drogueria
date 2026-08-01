package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type DashboardHandler struct {
	dashboardService service.DashboardService
}

func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetSummary cubre RF-057 + RF-058 + RF-059 + RF-060.
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	summary, err := h.dashboardService.GetSummary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el dashboard"})
		return
	}
	c.JSON(http.StatusOK, summary)
}
