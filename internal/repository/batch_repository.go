package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type BatchRepository interface {
	Create(batch *domain.Batch) error
	FindByID(id uuid.UUID) (*domain.Batch, error)
	FindByMedicine(medicineID uuid.UUID) ([]domain.Batch, error)
	IncreaseQuantity(batchID uuid.UUID, qty int) error
	SumStockByMedicine(medicineID uuid.UUID) (int, error)
	FindExpiringSoon(withinDays int) ([]domain.Batch, error)
	FindExpired() ([]domain.Batch, error)
}

type batchRepository struct {
	db *gorm.DB
}

func NewBatchRepository(db *gorm.DB) BatchRepository {
	return &batchRepository{db: db}
}

func (r *batchRepository) Create(batch *domain.Batch) error {
	return r.db.Create(batch).Error
}

func (r *batchRepository) FindByID(id uuid.UUID) (*domain.Batch, error) {
	var batch domain.Batch
	if err := r.db.First(&batch, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &batch, nil
}

// FindByMedicine cubre RF-018: todos los lotes de un medicamento, más
// recientes primero.
func (r *batchRepository) FindByMedicine(medicineID uuid.UUID) ([]domain.Batch, error) {
	var batches []domain.Batch
	err := r.db.Where("medicine_id = ?", medicineID).Order("expiration_date").Find(&batches).Error
	return batches, err
}

// IncreaseQuantity cubre RF-016: suma unidades a un lote ya existente
// (no reemplaza el valor, lo incrementa).
func (r *batchRepository) IncreaseQuantity(batchID uuid.UUID, qty int) error {
	return r.db.Model(&domain.Batch{}).Where("id = ?", batchID).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", qty)).Error
}

// SumStockByMedicine es la base de RF-021: el stock real de un medicamento
// es la suma de todos sus lotes con cantidad disponible.
func (r *batchRepository) SumStockByMedicine(medicineID uuid.UUID) (int, error) {
	var total int
	err := r.db.Model(&domain.Batch{}).
		Where("medicine_id = ? AND quantity > 0", medicineID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&total).Error
	return total, err
}

// FindExpiringSoon cubre RF-023: lotes cuya fecha de vencimiento cae
// dentro de los próximos `withinDays` días, sin incluir los ya vencidos.
func (r *batchRepository) FindExpiringSoon(withinDays int) ([]domain.Batch, error) {
	now := time.Now()
	limit := now.AddDate(0, 0, withinDays)
	var batches []domain.Batch
	err := r.db.Where("quantity > 0 AND expiration_date > ? AND expiration_date <= ?", now, limit).
		Order("expiration_date").Find(&batches).Error
	return batches, err
}

// FindExpired cubre RF-024: lotes cuya fecha de vencimiento ya pasó.
func (r *batchRepository) FindExpired() ([]domain.Batch, error) {
	var batches []domain.Batch
	err := r.db.Where("quantity > 0 AND expiration_date <= ?", time.Now()).
		Order("expiration_date").Find(&batches).Error
	return batches, err
}
