package dto

type SystemSettingsResponse struct {
	BusinessName             string `json:"business_name"`
	BusinessNit              string `json:"business_nit"`
	BusinessAddress          string `json:"business_address"`
	BusinessPhone            string `json:"business_phone"`
	BusinessEmail            string `json:"business_email"`
	LegalName                string `json:"legal_name"`
	TaxRegime                string `json:"tax_regime"`
	FiscalAddress            string `json:"fiscal_address"`
	ExpirationAlertDays      int    `json:"expiration_alert_days"`
	ExpirationAlertEnabled   bool   `json:"expiration_alert_enabled"`
	StockAlertEnabled        bool   `json:"stock_alert_enabled"`
	StockAlertFrequencyHours int    `json:"stock_alert_frequency_hours"`
	StockAlertStartTime      string `json:"stock_alert_start_time"`
	StockAlertEndTime        string `json:"stock_alert_end_time"`
	ReporteZHour             string `json:"reporte_z_hour"`
	LastBackupAt             string `json:"last_backup_at,omitempty"`
}

// UpdateBusinessInfoRequest cubre RF-079.
type UpdateBusinessInfoRequest struct {
	BusinessName    string `json:"business_name" binding:"required"`
	BusinessNit     string `json:"business_nit"`
	BusinessAddress string `json:"business_address"`
	BusinessPhone   string `json:"business_phone"`
	BusinessEmail   string `json:"business_email"`
}

// UpdateExpirationSettingsRequest cubre RF-080.
type UpdateExpirationSettingsRequest struct {
	ExpirationAlertDays    int  `json:"expiration_alert_days" binding:"gte=1"`
	ExpirationAlertEnabled bool `json:"expiration_alert_enabled"`
}

// UpdateStockSettingsRequest cubre RF-081.
type UpdateStockSettingsRequest struct {
	StockAlertEnabled        bool   `json:"stock_alert_enabled"`
	StockAlertFrequencyHours int    `json:"stock_alert_frequency_hours" binding:"gte=1"`
	StockAlertStartTime      string `json:"stock_alert_start_time" binding:"required"`
	StockAlertEndTime        string `json:"stock_alert_end_time" binding:"required"`
}

// UpdateReporteZScheduleRequest cubre RF-083.
type UpdateReporteZScheduleRequest struct {
	ReporteZHour string `json:"reporte_z_hour" binding:"required"`
}

// BackupResponse cubre RF-084.
type BackupResponse struct {
	FileName  string `json:"file_name"`
	SizeBytes int64  `json:"size_bytes"`
	CreatedAt string `json:"created_at"`
}
