package domain

import (
	"time"

	"github.com/google/uuid"
)

// CashMovement cubre RF-050: cada movimiento de dinero en la caja
// (apertura, ventas en efectivo/Nequi, cierre, ajustes), registrado
// automáticamente con fecha, hora y usuario.
// Los movimientos manuales (RF-018) tienen MovementCategory = "manual".
type CashMovement struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	CashRegisterID uuid.UUID `gorm:"type:uuid;not null;index"`
	// MovementType: apertura | venta_efectivo | venta_nequi | cierre | ajuste | ingreso_manual | egreso_manual
	MovementType string    `gorm:"not null"`
	Amount       float64   `gorm:"not null"`
	Description  string    // descripción automática generada por el sistema
	Reason       string    // RF-018: justificación obligatoria para movimientos manuales
	ReferenceID  *uuid.UUID `gorm:"type:uuid"`
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	// MovementCategory: "auto" para movimientos del sistema, "manual" para los del RF-018.
	// Permite filtrar solo los manuales en el arqueo sin depender del MovementType.
	MovementCategory string    `gorm:"not null;default:auto"`
	CreatedAt        time.Time
}

func (CashMovement) TableName() string { return "cash_movements" }
