package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type ReportHandler struct {
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{reportService: reportService}
}

// GetDailySalesReport cubre RF-061. GET /reports/daily-sales?date=AAAA-MM-DD
func (h *ReportHandler) GetDailySalesReport(c *gin.Context) {
	dateStr := c.Query("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha inválida, formato esperado AAAA-MM-DD"})
		return
	}

	report, err := h.reportService.GetDailySalesReport(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el reporte diario"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetMonthlySalesReport cubre RF-062. GET /reports/monthly-sales?year=2026&month=7
func (h *ReportHandler) GetMonthlySalesReport(c *gin.Context) {
	now := time.Now()
	year := now.Year()
	month := int(now.Month())

	if yStr := c.Query("year"); yStr != "" {
		if y, err := strconv.Atoi(yStr); err == nil {
			year = y
		}
	}
	if mStr := c.Query("month"); mStr != "" {
		if m, err := strconv.Atoi(mStr); err == nil {
			month = m
		}
	}

	report, err := h.reportService.GetMonthlySalesReport(year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el reporte mensual"})
		return
	}

	c.JSON(http.StatusOK, report)
}
