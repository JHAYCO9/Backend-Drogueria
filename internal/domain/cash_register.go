package domain

import (
	"time"

	"github.com/google/uuid"
)

type CashRegister struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	OpenedBy         uuid.UUID `gorm:"type:uuid;not null;index"` // index para búsqueda por usuario (RNF-008)
	ClosedBy         *uuid.UUID `gorm:"type:uuid"`
	OpeningAmount    float64  `gorm:"not null"`
	ExpectedAmount   *float64 // RF-052/054
	CountedAmount    *float64 // RF-052
	DifferenceAmount *float64 // RF-053
	DifferenceReason *string  // RF-053
	// RNF-011: clasificación del resultado del arqueo al cerrar.
	// Valores posibles: "EXCESS" (sobrante), "DEFICIT" (faltante), "NONE" (cuadra exacto).
	DiscrepancyType *string `gorm:"default:null"`
	Status          string  `gorm:"not null;default:abierta"`
	OpenedAt        time.Time
	ClosedAt        *time.Time
}

func (CashRegister) TableName() string { return "cash_registers" }
