package dto

// DailySalesReportResponse cubre RF-061.
type DailySalesReportResponse struct {
	Date          string  `json:"date"`
	SalesCount    int     `json:"sales_count"`
	TotalSold     float64 `json:"total_sold"`
	TotalEfectivo float64 `json:"total_efectivo"`
	TotalNequi    float64 `json:"total_nequi"`
	AverageTicket float64 `json:"average_ticket"`
}

// MonthlySalesReportResponse cubre RF-062.
type MonthlySalesReportResponse struct {
	Month        string  `json:"month"` // "2026-07"
	SalesCount   int     `json:"sales_count"`
	TotalSold    float64 `json:"total_sold"`
	AverageDaily float64 `json:"average_daily"`
	BestDay      string  `json:"best_day"`
	BestDayTotal float64 `json:"best_day_total"`
}

// TopSellingProductItem representa un producto en el reporte RF-063.
type TopSellingProductItem struct {
	MedicineID     string  `json:"medicine_id"`
	Name           string  `json:"name"`
	QuantitySold   int     `json:"quantity_sold"`
	TotalGenerated float64 `json:"total_generated"`
}

// TopSellingProductsResponse cubre RF-063: productos más vendidos,
// ordenados de mayor a menor por cantidad vendida.
type TopSellingProductsResponse struct {
	GeneratedAt string                  `json:"generated_at"`
	From        string                  `json:"from"`
	To          string                  `json:"to"`
	Products    []TopSellingProductItem  `json:"products"`
}

// LowRotationProductItem representa un producto en el reporte RF-064.
type LowRotationProductItem struct {
	MedicineID   string `json:"medicine_id"`
	Name         string `json:"name"`
	QuantitySold int    `json:"quantity_sold"`
}

// LowRotationProductsResponse cubre RF-064: productos con menor rotación,
// ordenados de menor a mayor por cantidad vendida.
type LowRotationProductsResponse struct {
	GeneratedAt string                    `json:"generated_at"`
	From        string                    `json:"from"`
	To          string                    `json:"to"`
	Products    []LowRotationProductItem  `json:"products"`
}
