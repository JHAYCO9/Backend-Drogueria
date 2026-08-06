package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type SaleExportHandler struct {
	exportService service.SaleExportService
}

func NewSaleExportHandler(exportService service.SaleExportService) *SaleExportHandler {
	return &SaleExportHandler{exportService: exportService}
}

// ExportSales cubre RF-087 + RF-088. GET /sales/export?from=AAAA-MM-DD&to=AAAA-MM-DD
func (h *SaleExportHandler) ExportSales(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "los parámetros from y to son obligatorios (formato AAAA-MM-DD)"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha 'from' inválida"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha 'to' inválida"})
		return
	}
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	result, err := h.exportService.ExportSales(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo exportar la información"})
		return
	}

	c.JSON(http.StatusOK, result)
}
