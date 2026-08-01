package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type ReporteZHandler struct {
	reporteZService service.ReporteZService
}

func NewReporteZHandler(reporteZService service.ReporteZService) *ReporteZHandler {
	return &ReporteZHandler{reporteZService: reporteZService}
}

// GetReporteZ cubre RF-055.
func (h *ReporteZHandler) GetReporteZ(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de caja inválido"})
		return
	}

	report, err := h.reporteZService.Generate(id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "no se pudo generar el reporte, verifica que la caja esté cerrada"})
		return
	}

	c.JSON(http.StatusOK, report)
}
