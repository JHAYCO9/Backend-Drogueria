package dto

// ─────────────────────────────────────────────────────────────
// RF-016: Apertura de caja
// ─────────────────────────────────────────────────────────────

type OpenCashRegisterRequest struct {
	OpeningAmount float64 `json:"opening_amount" binding:"required,gte=0"`
}

type OpenCashRegisterResponse struct {
	ID            string  `json:"id"`
	OpeningAmount float64 `json:"opening_amount"`
	OpenedAt      string  `json:"opened_at"`
}

// ─────────────────────────────────────────────────────────────
// RF-017: Cierre de caja con discrepancias
// ─────────────────────────────────────────────────────────────

// CloseCashRegisterRequest cubre RF-052 + RF-053: el cajero cuenta el
// efectivo físico y lo reporta; si hay diferencia, el motivo es opcional
// pero recomendado.
type CloseCashRegisterRequest struct {
	CountedAmount    float64 `json:"counted_amount" binding:"required,gte=0"`
	DifferenceReason string  `json:"difference_reason"`
}

// CloseCashRegisterResponse cubre RF-054 + RNF-011: totales calculados
// automáticamente, incluyendo la clasificación de la discrepancia.
type CloseCashRegisterResponse struct {
	ExpectedAmount   float64 `json:"expected_amount"`
	CountedAmount    float64 `json:"counted_amount"`
	DifferenceAmount float64 `json:"difference_amount"`
	// DiscrepancyType: "EXCESS" (sobrante), "DEFICIT" (faltante), "NONE" (cuadra).
	DiscrepancyType string `json:"discrepancy_type"`
	ClosedAt        string `json:"closed_at"`
}

// ─────────────────────────────────────────────────────────────
// RF-018: Movimientos manuales de caja
// ─────────────────────────────────────────────────────────────

// RegisterManualMovementRequest cubre RF-018: ingreso o egreso manual
// registrado explícitamente por el cajero durante la sesión.
type RegisterManualMovementRequest struct {
	// Type: "ingreso_manual" (dinero que entra) o "egreso_manual" (dinero que sale).
	Type   string  `json:"type" binding:"required,oneof=ingreso_manual egreso_manual"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
	Reason string  `json:"reason" binding:"required,min=3"`
}

type RegisterManualMovementResponse struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
	CreatedAt string  `json:"created_at"`
}

// ─────────────────────────────────────────────────────────────
// RF-019: Estado detallado de caja (arqueo en tiempo real)
// ─────────────────────────────────────────────────────────────

// ManualMovementItem representa un movimiento manual en la lista del arqueo.
type ManualMovementItem struct {
	ID        string  `json:"id"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Reason    string  `json:"reason"`
	CreatedAt string  `json:"created_at"`
}

// CashStatusDetailResponse cubre RF-019: arqueo completo con todos los
// totales desglosados y la lista de movimientos manuales de la sesión.
type CashStatusDetailResponse struct {
	IsOpen    bool   `json:"is_open"`
	SessionID string `json:"session_id,omitempty"`
	// Apertura
	MontoInicial  float64 `json:"monto_inicial,omitempty"`
	FechaApertura string  `json:"fecha_apertura,omitempty"`
	AbiertoPor    string  `json:"abierto_por,omitempty"`
	// Ventas de la sesión (solo finalizadas)
	TotalVentasEfectivo    float64 `json:"total_ventas_efectivo"`
	TotalVentasOtrosMedios float64 `json:"total_ventas_otros_medios"`
	// Movimientos manuales
	TotalIngresosManual float64 `json:"total_ingresos_manuales"`
	TotalEgresosManual  float64 `json:"total_egresos_manuales"`
	// Saldo teórico: MontoInicial + VentasEfectivo + Ingresos - Egresos
	SaldoTeorico float64 `json:"saldo_teorico_actual"`
	// Lista de movimientos manuales registrados en la sesión
	ListaMovimientos []ManualMovementItem `json:"lista_movimientos"`
}

// ─────────────────────────────────────────────────────────────
// RF-051 (existente): Movimientos de caja históricos
// ─────────────────────────────────────────────────────────────

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

type CashMovementResponse struct {
	ID               string  `json:"id"`
	MovementType     string  `json:"movement_type"`
	MovementCategory string  `json:"movement_category"`
	Amount           float64 `json:"amount"`
	Description      string  `json:"description"`
	Reason           string  `json:"reason,omitempty"`
	Username         string  `json:"username"`
	CreatedAt        string  `json:"created_at"`
}
