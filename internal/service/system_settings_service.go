package service

import (
	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type SystemSettingsService interface {
	Get() (*domain.SystemSettings, error)
	UpdateBusinessInfo(name, nit, address, phone, email string) error                      // RF-079
	UpdateExpirationSettings(days int, enabled bool) error                                 // RF-080
	UpdateStockSettings(enabled bool, frequencyHours int, startTime, endTime string) error // RF-081
	UpdateReporteZSchedule(hour string) error                                              // RF-083
	UpdateTaxSettings(nit, legalName, taxRegime, fiscalAddress string) error               // RF-086
}

type systemSettingsService struct {
	repo repository.SystemSettingsRepository
}

func NewSystemSettingsService(repo repository.SystemSettingsRepository) SystemSettingsService {
	return &systemSettingsService{repo: repo}
}

func (s *systemSettingsService) Get() (*domain.SystemSettings, error) {
	return s.repo.Get()
}

// UpdateBusinessInfo cubre RF-079. RNF-022: solo se tocan los campos
// de negocio, el resto de la configuración queda intacta.
func (s *systemSettingsService) UpdateBusinessInfo(name, nit, address, phone, email string) error {
	settings, err := s.repo.Get()
	if err != nil {
		return err
	}
	settings.BusinessName = name
	settings.BusinessNit = nit
	settings.BusinessAddress = address
	settings.BusinessPhone = phone
	settings.BusinessEmail = email
	return s.repo.Update(settings)
}

func (s *systemSettingsService) UpdateExpirationSettings(days int, enabled bool) error {
	settings, err := s.repo.Get()
	if err != nil {
		return err
	}
	settings.ExpirationAlertDays = days
	settings.ExpirationAlertEnabled = enabled
	return s.repo.Update(settings)
}

func (s *systemSettingsService) UpdateStockSettings(enabled bool, frequencyHours int, startTime, endTime string) error {
	settings, err := s.repo.Get()
	if err != nil {
		return err
	}
	settings.StockAlertEnabled = enabled
	settings.StockAlertFrequencyHours = frequencyHours
	settings.StockAlertStartTime = startTime
	settings.StockAlertEndTime = endTime
	return s.repo.Update(settings)
}

func (s *systemSettingsService) UpdateReporteZSchedule(hour string) error {
	settings, err := s.repo.Get()
	if err != nil {
		return err
	}
	settings.ReporteZHour = hour
	return s.repo.Update(settings)
}

// UpdateTaxSettings cubre RF-086. RNF-026: solo accesible desde rutas
// admin (validado en el router), y solo toca los campos tributarios.
func (s *systemSettingsService) UpdateTaxSettings(nit, legalName, taxRegime, fiscalAddress string) error {
	settings, err := s.repo.Get()
	if err != nil {
		return err
	}
	settings.BusinessNit = nit
	settings.LegalName = legalName
	settings.TaxRegime = taxRegime
	settings.FiscalAddress = fiscalAddress
	return s.repo.Update(settings)
}
