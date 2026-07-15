package domain

import (
	"time"

	"github.com/google/uuid"
)

// Batch representa un lote de un medicamento (RF-015). Cada lote tiene su
// propia fecha de vencimiento y cantidad — es la unidad real que se vende
// bajo la lógica FEFO (Sprint 3), no el medicamento en sí.
type Batch struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	MedicineID     uuid.UUID `gorm:"not null;index"`
	BatchCode      string    `gorm:"not null"`
	Quantity       int       `gorm:"not null;default:0"`
	ReceivedAt     time.Time `gorm:"not null"` // fecha de ingreso
	ExpirationDate time.Time `gorm:"not null"`
	CreatedAt      time.Time
}

func (Batch) TableName() string { return "batches" }
