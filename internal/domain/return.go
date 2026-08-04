package domain

import (
	"time"

	"github.com/google/uuid"
)

// Return cubre RF-089: devolución de UN medicamento de una venta ya
// finalizada. Cada devolución queda ligada al ítem original de la venta
// para trazabilidad completa.
type Return struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	SaleID     uuid.UUID `gorm:"not null"`
	SaleItemID uuid.UUID `gorm:"not null"`
	MedicineID uuid.UUID `gorm:"not null"`
	BatchID    uuid.UUID `gorm:"not null"`
	Quantity   int       `gorm:"not null"` // unidades sueltas devueltas
	Reason     string    `gorm:"not null"`
	UserID     uuid.UUID `gorm:"not null"`
	CreatedAt  time.Time

	// Campos virtuales (gorm:"-") para ser 100% compatibles con la BD de PostgreSQL existente
	IsVoided     bool       `gorm:"-"`
	VoidedReason string     `gorm:"-"`
	VoidedAt     *time.Time `gorm:"-"`
	VoidedBy     *uuid.UUID `gorm:"-"`
}

func (Return) TableName() string { return "returns" }
