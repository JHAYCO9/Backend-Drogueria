package domain

import (
	"time"

	"github.com/google/uuid"
)

type CashRegister struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	OpenedBy         uuid.UUID `gorm:"not null"`
	ClosedBy         *uuid.UUID
	OpeningAmount    float64  `gorm:"not null"`
	ExpectedAmount   *float64 // RF-052/054
	CountedAmount    *float64 // RF-052
	DifferenceAmount *float64 // RF-053
	DifferenceReason *string  // RF-053
	Status           string   `gorm:"not null;default:abierta"`
	OpenedAt         time.Time
	ClosedAt         *time.Time
}

func (CashRegister) TableName() string { return "cash_registers" }
