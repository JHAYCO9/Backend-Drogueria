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
	SaleUnit   string `json:"sale_unit" binding:"required,oneof=pastilla sello caja"`
}

type SaleItemResponse struct {
	ID           string  `json:"id"`
	MedicineID   string  `json:"medicine_id"`
	MedicineName string  `json:"medicine_name"`
	BatchID      string  `json:"batch_id"`
	BatchCode    string  `json:"batch_code"`
	Quantity     int     `json:"quantity"`
	SaleUnit     string  `json:"sale_unit"`
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

// SaleListItem cubre RF-044: vista resumida para el historial de ventas.
type SaleListItem struct {
	ID            string  `json:"id"`
	SaleNumber    int     `json:"sale_number"`
	Status        string  `json:"status"`
	TotalAmount   float64 `json:"total_amount"`
	PaymentMethod *string `json:"payment_method"`
	CashierName   string  `json:"cashier_name"`
	CreatedAt     string  `json:"created_at"`
}

// SalesDateRangeSummary cubre RF-047: número de ventas, total vendido y
// promedio por venta en el rango de fechas consultado.
type SalesDateRangeSummary struct {
	FromDate    string         `json:"from_date"`
	ToDate      string         `json:"to_date"`
	SalesCount  int            `json:"sales_count"`
	TotalSold   float64        `json:"total_sold"`
	AverageSale float64        `json:"average_sale"`
	Sales       []SaleListItem `json:"sales"`
}

// CreateSaleRequest es el body para la venta atómica (una sola llamada HTTP).
type CreateSaleItemRequest struct {
	MedicineID string `json:"medicine_id" binding:"required"`
	BatchID    string `json:"batch_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
	SaleUnit   string `json:"sale_unit" binding:"required,oneof=pastilla sello caja"`
}

type CreateSaleRequest struct {
	CustomerID    *string                 `json:"customer_id"`
	PaymentMethod string                  `json:"payment_method" binding:"required,oneof=efectivo tarjeta transferencia nequi"`
	AmountPaid    float64                 `json:"amount_paid" binding:"required,gt=0"`
	Items         []CreateSaleItemRequest `json:"items" binding:"required,min=1,dive"`
}

// AtomicSaleItemResponse es el detalle de cada ítem en la respuesta de venta atómica.
type AtomicSaleItemResponse struct {
	MedicineID   string  `json:"medicine_id"`
	MedicineName string  `json:"medicine_name"`
	BatchID      string  `json:"batch_id"`
	BatchCode    string  `json:"batch_code"`
	Quantity     int     `json:"quantity"`
	SaleUnit     string  `json:"sale_unit"`
	UnitPrice    float64 `json:"unit_price"`
	Subtotal     float64 `json:"subtotal"`
	TaxAmount    float64 `json:"tax_amount"`
}

// AtomicSaleResponse es la respuesta completa de la venta atómica finalizada.
type AtomicSaleResponse struct {
	ID            string                   `json:"id"`
	SaleNumber    int                      `json:"sale_number"`
	CustomerID    *string                  `json:"customer_id"`
	CustomerName  string                   `json:"customer_name"`
	CashierName   string                   `json:"cashier_name"`
	Items         []AtomicSaleItemResponse `json:"items"`
	Subtotal      float64                  `json:"subtotal"`
	TotalTax      float64                  `json:"total_tax"`
	TotalAmount   float64                  `json:"total_amount"`
	PaymentMethod string                   `json:"payment_method"`
	AmountPaid    float64                  `json:"amount_paid"`
	Change        float64                  `json:"change"`
	Status        string                   `json:"status"`
	CreatedAt     string                   `json:"created_at"`
}

// CancelSaleRequest es el body para anular una venta completada (RF-046).
type CancelSaleRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}
