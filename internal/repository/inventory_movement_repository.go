package repository

import (
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type InventoryMovementRepository interface {
	Create(movement *domain.InventoryMovement) error
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
