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
