package dto

// SaleExportItem cubre RF-087: estructura compatible con facturación
// electrónica a nivel de línea de producto.
type SaleExportItem struct {
	MedicineName string  `json:"medicine_name"`
	Quantity     int     `json:"quantity"`
	SaleUnit     string  `json:"sale_unit"`
	UnitPrice    float64 `json:"unit_price"`
	BaseAmount   float64 `json:"base_amount"`
	TaxAmount    float64 `json:"tax_amount"`
	Subtotal     float64 `json:"subtotal"`
}

// SaleExport cubre RF-087: la venta completa con todos los campos que
// exige una futura integración de facturación electrónica (consecutivo,
// fecha, productos, cantidades, impuestos, método de pago, totales).
type SaleExport struct {
	Consecutive   int              `json:"consecutive"`
	Date          string           `json:"date"`
	Items         []SaleExportItem `json:"items"`
	TotalTax      float64          `json:"total_tax"`
	PaymentMethod string           `json:"payment_method"`
	TotalAmount   float64          `json:"total_amount"`
	CashierName   string           `json:"cashier_name"`
}

// SalesExportResponse cubre RF-088.
type SalesExportResponse struct {
	ExportedAt string       `json:"exported_at"`
	FromDate   string       `json:"from_date"`
	ToDate     string       `json:"to_date"`
	Count      int          `json:"count"`
	Sales      []SaleExport `json:"sales"`
}
