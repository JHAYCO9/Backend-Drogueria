package domain

import (
	"time"

	"github.com/google/uuid"
)

// InventoryMovement es el historial de todo cambio de stock (RF-016 en
// adelante). No se edita ni se borra nunca — es el rastro de auditoría
// del inventario.
type InventoryMovement struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	MedicineID   uuid.UUID `gorm:"not null;index"`
	BatchID      *uuid.UUID
	MovementType string `gorm:"not null"` // "ingreso", "ajuste", "venta", "devolucion"
	Quantity     int    `gorm:"not null"`
	Reason       string
	UserID       uuid.UUID `gorm:"not null"`
	ReferenceID  *uuid.UUID
	CreatedAt    time.Time
}

func (InventoryMovement) TableName() string { return "inventory_movements" }
