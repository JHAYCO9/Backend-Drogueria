package domain

import (
	"time"

	"github.com/google/uuid"
)

// SystemSettings cubre RF-079/080/081/083/084: configuración global
// del sistema, siempre una única fila.
type SystemSettings struct {
	ID                       uuid.UUID `gorm:"type:uuid;primaryKey"`
	BusinessName             string
	BusinessNit              string
	BusinessAddress          string
	BusinessPhone            string
	BusinessEmail            string
	LegalName                string // RF-086
	TaxRegime                string // RF-086
	FiscalAddress            string // RF-086
	ExpirationAlertDays      int
	ExpirationAlertEnabled   bool
	StockAlertEnabled        bool
	StockAlertFrequencyHours int
	StockAlertStartTime      string
	StockAlertEndTime        string
	ReporteZHour             string
	LastBackupAt             *time.Time
}

func (SystemSettings) TableName() string { return "system_settings" }
