package repository

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type SyncRepository interface {
	Create(event *domain.SyncEvent) (created bool, err error)
}

type syncRepository struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) SyncRepository {
	return &syncRepository{db: db}
}

// Create inserta el evento de forma idempotente: si ya existe un registro
// con este mismo ID (porque el cliente lo reenvió tras un timeout), no
// hace nada y no falla — simplemente confirma que ya estaba procesado.
// Esto es justo el criterio de aceptación de "reenviar el mismo evento
// dos veces no genera duplicados".
func (r *syncRepository) Create(event *domain.SyncEvent) (bool, error) {
	result := r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
