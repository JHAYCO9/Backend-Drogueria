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

// parseReportDateRange extrae los query params from/to comunes a RF-063 y RF-064.
// Si no se proporcionan, usa el mes actual completo como rango por defecto.
func parseReportDateRange(c *gin.Context) (time.Time, time.Time, error) {
	now := time.Now()

	fromStr := c.Query("from")
	toStr := c.Query("to")

	var from, to time.Time

	if fromStr == "" {
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	} else {
		var err error
		from, err = time.Parse("2006-01-02", fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	if toStr == "" {
		to = from.AddDate(0, 1, 0).Add(-time.Second)
	} else {
		var err error
		to, err = time.Parse("2006-01-02", toStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	return from, to, nil
}

// GetTopSellingProducts cubre RF-063. GET /reports/top-selling?from=AAAA-MM-DD&to=AAAA-MM-DD&limit=10
func (h *ReportHandler) GetTopSellingProducts(c *gin.Context) {
	from, to, err := parseReportDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha inválida, formato esperado AAAA-MM-DD"})
		return
	}

	limit := 10
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	report, err := h.reportService.GetTopSellingProducts(from, to, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el reporte de productos más vendidos"})
		return
	}

	c.JSON(http.StatusOK, report)
}

// GetLowRotationProducts cubre RF-064. GET /reports/low-rotation?from=AAAA-MM-DD&to=AAAA-MM-DD&limit=10
func (h *ReportHandler) GetLowRotationProducts(c *gin.Context) {
	from, to, err := parseReportDateRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha inválida, formato esperado AAAA-MM-DD"})
		return
	}

	limit := 10
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	report, err := h.reportService.GetLowRotationProducts(from, to, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo generar el reporte de productos con menor rotación"})
		return
	}

	c.JSON(http.StatusOK, report)
}
