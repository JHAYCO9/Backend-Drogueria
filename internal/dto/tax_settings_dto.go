package dto

// UpdateTaxSettingsRequest cubre RF-086.
type UpdateTaxSettingsRequest struct {
	Nit           string `json:"nit" binding:"required"`
	LegalName     string `json:"legal_name" binding:"required"`
	TaxRegime     string `json:"tax_regime" binding:"required"`
	FiscalAddress string `json:"fiscal_address" binding:"required"`
}

type TaxSettingsResponse struct {
	Nit           string `json:"nit"`
	LegalName     string `json:"legal_name"`
	TaxRegime     string `json:"tax_regime"`
	FiscalAddress string `json:"fiscal_address"`
}
