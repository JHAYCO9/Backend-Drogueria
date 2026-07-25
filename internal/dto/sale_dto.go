package dto

type StartSaleResponse struct {
	ID         string `json:"id"`
	SaleNumber int    `json:"sale_number"`
	CreatedAt  string `json:"created_at"`
	UserName   string `json:"user_name"`
}

// AddSaleItemRequest cubre RF-033. Acepta MedicineID directo (si ya se
// buscó por nombre en el frontend) o Barcode (si viene del lector físico,
// RF-027) — al menos uno de los dos es obligatorio, se valida en el service.
type AddSaleItemRequest struct {
	MedicineID string `json:"medicine_id"`
	Barcode    string `json:"barcode"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
}

type SaleItemResponse struct {
	ID           string  `json:"id"`
	MedicineID   string  `json:"medicine_id"`
	MedicineName string  `json:"medicine_name"`
	BatchID      string  `json:"batch_id"`
	BatchCode    string  `json:"batch_code"`
	Quantity     int     `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	Subtotal     float64 `json:"subtotal"`
	TaxAmount    float64 `json:"tax_amount"`
}

type SaleResponse struct {
	ID             string             `json:"id"`
	SaleNumber     int                `json:"sale_number"`
	Status         string             `json:"status"`
	Items          []SaleItemResponse `json:"items"`
	TotalAmount    float64            `json:"total_amount"`
	TotalTaxAmount float64            `json:"total_tax_amount"`
	PaymentMethod  *string            `json:"payment_method"`
	AmountReceived *float64           `json:"amount_received"`
	ChangeGiven    *float64           `json:"change_given"`
	CreatedAt      string             `json:"created_at"`
	FinalizedAt    *string            `json:"finalized_at"`
}

// FinalizeCashRequest cubre RF-039.
type FinalizeCashRequest struct {
	AmountReceived float64 `json:"amount_received" binding:"required,gt=0"`
}

// FinalizeNequiRequest cubre RF-040.
type FinalizeNequiRequest struct {
	AmountPaid float64 `json:"amount_paid" binding:"required,gt=0"`
}
