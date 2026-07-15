package domain

import (
	"time"

	"github.com/google/uuid"
)

// Medicine representa un medicamento comercializable (RF-011).
// El precio se maneja en tres presentaciones distintas (pastilla, sello,
// caja), tal como lo pide el requerimiento, en vez de un único precio.
type Medicine struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	Barcode             string    `gorm:"uniqueIndex;not null"`
	Name                string    `gorm:"not null"`
	Laboratory          string
	Concentration       string
	Presentation        string
	PricePerPill        float64 // precio por pastilla
	PricePerSeal        float64 // precio por sello (blister/tira)
	PricePerBox         float64 // precio por caja completa
	MinStock            int     `gorm:"not null;default:0"`  // RF-020
	ExpirationAlertDays int     `gorm:"not null;default:30"` // RF-022: días antes de vencer para alertar
	IsActive            bool    `gorm:"default:true"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (Medicine) TableName() string { return "medicines" }
