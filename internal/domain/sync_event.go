package domain

import (
	"time"

	"github.com/google/uuid"
)

// SyncEvent es el espejo del outbox del cliente. El ID lo genera el
// cliente (no el servidor) — es la clave de la idempotencia: si el mismo
// evento se reenvía por un timeout de red, el servidor lo reconoce por
// este mismo ID y no lo duplica.
type SyncEvent struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	EntityType      string
	EntityID        uuid.UUID
	Operation       string
	Payload         string `gorm:"type:jsonb"`
	Status          string `gorm:"default:recibido"`
	ErrorDetail     *string
	ClientCreatedAt time.Time
	ReceivedAt      time.Time
	ProcessedAt     *time.Time
}

func (SyncEvent) TableName() string { return "sync_events" }
