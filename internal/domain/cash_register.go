package domain

import (
	"time"

	"github.com/google/uuid"
)

// CashRegister es el registro mínimo de apertura/cierre de caja, suficiente
// para que Ventas (RF-032) valide que existe una jornada activa. El resto
// de Caja (arqueo, cierre, Reporte Z) se completa en su propio requerimiento
// más adelante, sin romper lo que se construye aquí.
type CashRegister struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	OpenedBy      uuid.UUID `gorm:"not null"`
	ClosedBy      *uuid.UUID
	OpeningAmount float64 `gorm:"not null"`
	Status        string  `gorm:"not null;default:abierta"` // "abierta" | "cerrada"
	OpenedAt      time.Time
	ClosedAt      *time.Time
}

func (CashRegister) TableName() string { return "cash_registers" }