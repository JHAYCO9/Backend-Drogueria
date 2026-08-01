package dto

type OpenCashRegisterRequest struct {
	OpeningAmount float64 `json:"opening_amount" binding:"required,gte=0"`
}

type OpenCashRegisterResponse struct {
	ID            string  `json:"id"`
	OpeningAmount float64 `json:"opening_amount"`
	OpenedAt      string  `json:"opened_at"`
}

type CashRegisterStatusResponse struct {
	IsOpen          bool    `json:"is_open"`
	ID              string  `json:"id,omitempty"`
	OpeningAmount   float64 `json:"opening_amount,omitempty"`
	ExpectedCash    float64 `json:"expected_cash,omitempty"`
	TotalNequiSales float64 `json:"total_nequi_sales,omitempty"`
	CurrentBalance  float64 `json:"current_balance,omitempty"`
	OpenedAt        string  `json:"opened_at,omitempty"`
	OpenedByName    string  `json:"opened_by_name,omitempty"`
}

// CloseCashRegisterRequest cubre RF-052 + RF-053: el cajero cuenta el
// efectivo físico y lo reporta; si hay diferencia, el motivo es obligatorio.
type CloseCashRegisterRequest struct {
	CountedAmount    float64 `json:"counted_amount" binding:"required,gte=0"`
	DifferenceReason string  `json:"difference_reason"`
}

// CloseCashRegisterResponse cubre RF-054: totales calculados automáticamente.
type CloseCashRegisterResponse struct {
	ExpectedAmount   float64 `json:"expected_amount"`
	CountedAmount    float64 `json:"counted_amount"`
	DifferenceAmount float64 `json:"difference_amount"`
	ClosedAt         string  `json:"closed_at"`
}

type CashMovementResponse struct {
	ID           string  `json:"id"`
	MovementType string  `json:"movement_type"`
	Amount       float64 `json:"amount"`
	Description  string  `json:"description"`
	Username     string  `json:"username"`
	CreatedAt    string  `json:"created_at"`
}
