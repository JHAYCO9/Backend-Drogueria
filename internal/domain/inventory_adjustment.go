package domain

import (
	"time"

	"github.com/google/uuid"
)

// InventoryAdjustment almacena el detalle completo de cada ajuste manual
// de inventario (RF-017). Se registra en tabla separada de movements
// porque necesita campos adicionales (cantidad anterior, nueva, motivo).
type InventoryAdjustment struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	MedicineID       uuid.UUID `gorm:"not null;index"`
	BatchID          uuid.UUID `gorm:"not null;index"`
	PreviousQuantity int       `gorm:"not null"`
	NewQuantity      int       `gorm:"not null"`
	Reason           string    `gorm:"not null"`
	UserID           uuid.UUID `gorm:"not null"`
	CreatedAt        time.Time
}

func (InventoryAdjustment) TableName() string { return "inventory_adjustments" }
