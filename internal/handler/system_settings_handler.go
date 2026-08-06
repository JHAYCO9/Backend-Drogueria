package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type SystemSettingsHandler struct {
	settingsService service.SystemSettingsService
	backupService   service.BackupService
}

func NewSystemSettingsHandler(settingsService service.SystemSettingsService, backupService service.BackupService) *SystemSettingsHandler {
	return &SystemSettingsHandler{settingsService: settingsService, backupService: backupService}
}

func (h *SystemSettingsHandler) Get(c *gin.Context) {
	s, err := h.settingsService.Get()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar la configuración"})
		return
	}

	response := dto.SystemSettingsResponse{
		BusinessName: s.BusinessName, BusinessNit: s.BusinessNit, BusinessAddress: s.BusinessAddress,
		BusinessPhone: s.BusinessPhone, BusinessEmail: s.BusinessEmail,
		LegalName: s.LegalName, TaxRegime: s.TaxRegime, FiscalAddress: s.FiscalAddress,
		ExpirationAlertDays: s.ExpirationAlertDays, ExpirationAlertEnabled: s.ExpirationAlertEnabled,
		StockAlertEnabled: s.StockAlertEnabled, StockAlertFrequencyHours: s.StockAlertFrequencyHours,
		StockAlertStartTime: s.StockAlertStartTime, StockAlertEndTime: s.StockAlertEndTime,
		ReporteZHour: s.ReporteZHour,
	}
	if s.LastBackupAt != nil {
		response.LastBackupAt = s.LastBackupAt.Format(time.RFC3339)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateBusinessInfo cubre RF-079.
func (h *SystemSettingsHandler) UpdateBusinessInfo(c *gin.Context) {
	var req dto.UpdateBusinessInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el nombre de la droguería es obligatorio"})
		return
	}

	if err := h.settingsService.UpdateBusinessInfo(req.BusinessName, req.BusinessNit, req.BusinessAddress, req.BusinessPhone, req.BusinessEmail); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la información"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "información actualizada correctamente"})
}

// UpdateExpirationSettings cubre RF-080.
func (h *SystemSettingsHandler) UpdateExpirationSettings(c *gin.Context) {
	var req dto.UpdateExpirationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "los días de anticipación deben ser mayor o igual a 1"})
		return
	}

	if err := h.settingsService.UpdateExpirationSettings(req.ExpirationAlertDays, req.ExpirationAlertEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la configuración"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "configuración actualizada correctamente"})
}

// UpdateStockSettings cubre RF-081.
func (h *SystemSettingsHandler) UpdateStockSettings(c *gin.Context) {
	var req dto.UpdateStockSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	if err := h.settingsService.UpdateStockSettings(req.StockAlertEnabled, req.StockAlertFrequencyHours, req.StockAlertStartTime, req.StockAlertEndTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la configuración"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "configuración actualizada correctamente"})
}

// UpdateReporteZSchedule cubre RF-083.
func (h *SystemSettingsHandler) UpdateReporteZSchedule(c *gin.Context) {
	var req dto.UpdateReporteZScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la hora es obligatoria"})
		return
	}

	if err := h.settingsService.UpdateReporteZSchedule(req.ReporteZHour); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar el horario"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "horario actualizado correctamente"})
}

// CreateBackup cubre RF-084.
func (h *SystemSettingsHandler) CreateBackup(c *gin.Context) {
	fileName, size, err := h.backupService.CreateBackup()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo crear el respaldo: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dto.BackupResponse{
		FileName: fileName, SizeBytes: size, CreatedAt: time.Now().Format(time.RFC3339),
	})
}

// UpdateTaxSettings cubre RF-086.
func (h *SystemSettingsHandler) UpdateTaxSettings(c *gin.Context) {
	var req dto.UpdateTaxSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "todos los campos tributarios son obligatorios"})
		return
	}

	if err := h.settingsService.UpdateTaxSettings(req.Nit, req.LegalName, req.TaxRegime, req.FiscalAddress); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la información tributaria"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "información tributaria actualizada correctamente"})
}
