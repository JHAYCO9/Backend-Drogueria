package domain

import (
	"time"

	"github.com/google/uuid"
)

// Sale representa una venta. Nace "en_progreso" (RF-031) con el detalle
// vacío, se le van agregando SaleItems (RF-033/034/035) sin tocar el
// inventario todavía, y solo al finalizar (RF-041) se descuenta stock real.
type Sale struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SaleNumber     int        `gorm:"not null;autoIncrement"`
	CustomerID     *uuid.UUID `gorm:"type:uuid;index"` // nullable — Consumidor Final si no se especifica
	UserID         uuid.UUID  `gorm:"not null"`
	CashRegisterID uuid.UUID  `gorm:"not null"`
	Status         string     `gorm:"not null;default:en_progreso"`
	PaymentMethod  *string
	TotalAmount    float64  `gorm:"not null;default:0"`
	AmountReceived *float64
	ChangeGiven    *float64
	CancelledAt    *time.Time
	CancelledBy    *uuid.UUID `gorm:"type:uuid"` // ID del admin que anuló
	CancelReason   *string
	CreatedAt      time.Time
	FinalizedAt    *time.Time
}

func (Sale) TableName() string { return "sales" }

// SaleItem es cada línea del detalle de una venta (RF-033/035). El lote
// se asigna automáticamente por FEFO al momento de agregar el medicamento,
// pero el inventario NO se descuenta hasta que la venta se finalice.
// SaleUnit indica en qué presentación se vendió (pastilla/sello/caja) y
// UnitsConsumed es cuántas unidades sueltas representa realmente esa
// cantidad — es lo que se descuenta del inventario, que siempre se mide
// en la unidad más pequeña.
type SaleItem struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	SaleID        uuid.UUID `gorm:"not null;index"`
	MedicineID    uuid.UUID `gorm:"not null"`
	BatchID       uuid.UUID `gorm:"not null"`
	Quantity      int       `gorm:"not null"`
	SaleUnit      string    `gorm:"not null;default:pastilla"`
	UnitsConsumed int       `gorm:"not null;default:0"`
	UnitPrice     float64   `gorm:"not null"`
	Subtotal      float64   `gorm:"not null"`
	TaxAmount     float64   `gorm:"not null;default:0"`
	CreatedAt     time.Time
}

func (SaleItem) TableName() string { return "sale_items" }
