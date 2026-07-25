package domain

import (
	"time"

	"github.com/google/uuid"
)

// Sale representa una venta. Nace "en_progreso" (RF-031) con el detalle
// vacío, se le van agregando SaleItems (RF-033/034/035) sin tocar el
// inventario todavía, y solo al finalizar (RF-041) se descuenta stock real.
type Sale struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	SaleNumber     int       `gorm:"not null;autoIncrement"` // consecutivo, RF-031
	UserID         uuid.UUID `gorm:"not null"`
	CashRegisterID uuid.UUID `gorm:"not null"`
	Status         string    `gorm:"not null;default:en_progreso"` // en_progreso | finalizada | cancelada
	PaymentMethod  *string   // "efectivo" | "nequi", solo se llena al finalizar
	TotalAmount    float64   `gorm:"not null;default:0"`
	AmountReceived *float64  // RF-039: valor recibido en efectivo
	ChangeGiven    *float64  // RF-039: cambio entregado
	CreatedAt      time.Time
	FinalizedAt    *time.Time
}

func (Sale) TableName() string { return "sales" }

// SaleItem es cada línea del detalle de una venta (RF-033/035). El lote
// se asigna automáticamente por FEFO al momento de agregar el medicamento,
// pero el inventario NO se descuenta hasta que la venta se finalice.
type SaleItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	SaleID     uuid.UUID `gorm:"not null;index"`
	MedicineID uuid.UUID `gorm:"not null"`
	BatchID    uuid.UUID `gorm:"not null"`
	Quantity   int       `gorm:"not null"`
	UnitPrice  float64   `gorm:"not null"`
	Subtotal   float64   `gorm:"not null"`
	TaxAmount  float64   `gorm:"not null;default:0"` // porción de IVA incluida en el subtotal
	CreatedAt  time.Time
}

func (SaleItem) TableName() string { return "sale_items" }
