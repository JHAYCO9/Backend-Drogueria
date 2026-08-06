package repository

import (
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type SystemSettingsRepository interface {
	Get() (*domain.SystemSettings, error)
	Update(s *domain.SystemSettings) error
}

type systemSettingsRepository struct {
	db *gorm.DB
}

func NewSystemSettingsRepository(db *gorm.DB) SystemSettingsRepository {
	return &systemSettingsRepository{db: db}
}

func (r *systemSettingsRepository) Get() (*domain.SystemSettings, error) {
	var s domain.SystemSettings
	err := r.db.First(&s).Error
	return &s, err
}

// Update usa un mapa explícito de columnas (RNF-022: cambios íntegros,
// nunca se sobreescriben campos que no vienen en la petición).
func (r *systemSettingsRepository) Update(s *domain.SystemSettings) error {
	return r.db.Model(&domain.SystemSettings{}).Where("id = ?", s.ID).
		Updates(map[string]interface{}{
			"business_name":               s.BusinessName,
			"business_nit":                s.BusinessNit,
			"business_address":            s.BusinessAddress,
			"business_phone":              s.BusinessPhone,
			"business_email":              s.BusinessEmail,
			"legal_name":                  s.LegalName,
			"tax_regime":                  s.TaxRegime,
			"fiscal_address":              s.FiscalAddress,
			"expiration_alert_days":       s.ExpirationAlertDays,
			"expiration_alert_enabled":    s.ExpirationAlertEnabled,
			"stock_alert_enabled":         s.StockAlertEnabled,
			"stock_alert_frequency_hours": s.StockAlertFrequencyHours,
			"stock_alert_start_time":      s.StockAlertStartTime,
			"stock_alert_end_time":        s.StockAlertEndTime,
			"reporte_z_hour":              s.ReporteZHour,
		}).Error
}
