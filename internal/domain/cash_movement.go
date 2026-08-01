package domain

import (
	"time"

	"github.com/google/uuid"
)

// CashMovement cubre RF-050: cada movimiento de dinero en la caja
// (apertura, ventas en efectivo/Nequi, cierre, ajustes), registrado
// automáticamente con fecha, hora y usuario.
type CashMovement struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	CashRegisterID uuid.UUID `gorm:"not null"`
	MovementType   string    `gorm:"not null"` // apertura | venta_efectivo | venta_nequi | cierre | ajuste
	Amount         float64   `gorm:"not null"`
	Description    string
	ReferenceID    *uuid.UUID
	UserID         uuid.UUID `gorm:"not null"`
	CreatedAt      time.Time
}

func (CashMovement) TableName() string { return "cash_movements" }
