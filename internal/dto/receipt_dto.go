package dto

type ReceiptItemLine struct {
	MedicineName string  `json:"medicine_name"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	BasePrice    float64 `json:"base_price"` // precio sin IVA de la línea
	TaxAmount    float64 `json:"tax_amount"` // porción de IVA de la línea
	HasIva       bool    `json:"has_iva"`
	Subtotal     float64 `json:"subtotal"`
}

// ReceiptResponse cubre RF-043: todo lo que necesita el comprobante impreso.
type ReceiptResponse struct {
	SaleNumber     int               `json:"sale_number"`
	Date           string            `json:"date"` // "2026-07-24"
	Time           string            `json:"time"` // "16:40"
	CashierName    string            `json:"cashier_name"`
	Items          []ReceiptItemLine `json:"items"`
	TotalAmount    float64           `json:"total_amount"`
	TotalTax       float64           `json:"total_tax"`
	PaymentMethod  string            `json:"payment_method"`
	AmountReceived *float64          `json:"amount_received"`
	ChangeGiven    *float64          `json:"change_given"`
}
