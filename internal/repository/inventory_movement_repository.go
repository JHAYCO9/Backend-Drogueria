package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type InventoryMovementRepository interface {
	Create(movement *domain.InventoryMovement) error
	FindAll(movementType string, from, to *time.Time) ([]domain.InventoryMovement, error)
	FindByDateRange(from, to *time.Time, movementType string) ([]domain.InventoryMovement, error)
}

type inventoryMovementRepository struct {
	db *gorm.DB
}

func NewInventoryMovementRepository(db *gorm.DB) InventoryMovementRepository {
	return &inventoryMovementRepository{db: db}
}

func (r *inventoryMovementRepository) Create(movement *domain.InventoryMovement) error {
	return r.db.Create(movement).Error
}

// FindAll cubre RF-019: historial de movimientos con filtros opcionales
// por tipo y rango de fechas.
func (r *inventoryMovementRepository) FindAll(movementType string, from, to *time.Time) ([]domain.InventoryMovement, error) {
	var movements []domain.InventoryMovement
	query := r.db.Model(&domain.InventoryMovement{})

	if movementType != "" {
		query = query.Where("movement_type = ?", movementType)
	}
	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at <= ?", *to)
	}

	err := query.Order("created_at DESC").Find(&movements).Error
	return movements, err
}

// FindByDateRange cubre RF-030: movimientos filtrados por rango de
// fechas y tipo para el historial de existencias.
func (r *inventoryMovementRepository) FindByDateRange(from, to *time.Time, movementType string) ([]domain.InventoryMovement, error) {
	var movements []domain.InventoryMovement
	query := r.db.Model(&domain.InventoryMovement{})

	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at <= ?", *to)
	}
	if movementType != "" {
		query = query.Where("movement_type = ?", movementType)
	}

	err := query.Order("created_at DESC").Find(&movements).Error
	return movements, err
}
