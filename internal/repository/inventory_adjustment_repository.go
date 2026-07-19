package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type InventoryAdjustmentRepository interface {
	Create(adjustment *domain.InventoryAdjustment) error
	FindAll() ([]domain.InventoryAdjustment, error)
	FindByMedicine(medicineID uuid.UUID) ([]domain.InventoryAdjustment, error)
}

type inventoryAdjustmentRepository struct {
	db *gorm.DB
}

func NewInventoryAdjustmentRepository(db *gorm.DB) InventoryAdjustmentRepository {
	return &inventoryAdjustmentRepository{db: db}
}

func (r *inventoryAdjustmentRepository) Create(adjustment *domain.InventoryAdjustment) error {
	return r.db.Create(adjustment).Error
}

func (r *inventoryAdjustmentRepository) FindAll() ([]domain.InventoryAdjustment, error) {
	var adjustments []domain.InventoryAdjustment
	err := r.db.Order("created_at DESC").Find(&adjustments).Error
	return adjustments, err
}

func (r *inventoryAdjustmentRepository) FindByMedicine(medicineID uuid.UUID) ([]domain.InventoryAdjustment, error) {
	var adjustments []domain.InventoryAdjustment
	err := r.db.Where("medicine_id = ?", medicineID).Order("created_at DESC").Find(&adjustments).Error
	return adjustments, err
}
