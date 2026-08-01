package dto

// DashboardSummaryResponse cubre RF-057 + RF-058 + RF-059: todo lo que
// necesita el dashboard principal en una sola llamada, para no hacer
// múltiples round-trips al abrir la app.
type DashboardSummaryResponse struct {
	// RF-058: indicadores de ventas de hoy
	TodaySalesTotal       float64 `json:"today_sales_total"`
	TodaySalesCount       int     `json:"today_sales_count"`
	AverageTicket         float64 `json:"average_ticket"`
	MostUsedPaymentMethod string  `json:"most_used_payment_method"`

	// RF-059: indicadores de inventario
	LowStockCount     int `json:"low_stock_count"`
	OutOfStockCount   int `json:"out_of_stock_count"`
	ExpiringSoonCount int `json:"expiring_soon_count"`
	ExpiredCount      int `json:"expired_count"`

	// RF-057: estado de la caja
	CashRegisterOpen bool    `json:"cash_register_open"`
	CashBalance      float64 `json:"cash_balance,omitempty"`

	// RF-060: actividad reciente
	RecentActivity []ActivityItem `json:"recent_activity"`
}

type ActivityItem struct {
	Type        string  `json:"type"` // venta | ajuste | apertura_caja | cierre_caja
	Description string  `json:"description"`
	Amount      float64 `json:"amount,omitempty"`
	Username    string  `json:"username"`
	CreatedAt   string  `json:"created_at"`
}
