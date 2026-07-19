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
	UpdateQuantity(batchID uuid.UUID, newQty int) error
	FindAvailableByMedicineFEFO(medicineID uuid.UUID) ([]domain.Batch, error)
	FindEarliestExpiration(medicineID uuid.UUID) (*domain.Batch, error)
	CountAvailableByMedicine(medicineID uuid.UUID) (int, error)
	DecreaseQuantity(batchID uuid.UUID, qty int) error
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

// FindByMedicine cubre RF-018: todos los lotes de un medicamento, ordenados
// por fecha de vencimiento.
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
// dentro de los proximos `withinDays` dias, sin incluir los ya vencidos.
func (r *batchRepository) FindExpiringSoon(withinDays int) ([]domain.Batch, error) {
	now := time.Now()
	limit := now.AddDate(0, 0, withinDays)
	var batches []domain.Batch
	err := r.db.Where("quantity > 0 AND expiration_date > ? AND expiration_date <= ?", now, limit).
		Order("expiration_date").Find(&batches).Error
	return batches, err
}

// FindExpired cubre RF-024: lotes cuya fecha de vencimiento ya paso.
func (r *batchRepository) FindExpired() ([]domain.Batch, error) {
	var batches []domain.Batch
	err := r.db.Where("quantity > 0 AND expiration_date <= ?", time.Now()).
		Order("expiration_date").Find(&batches).Error
	return batches, err
}

// UpdateQuantity cubre RF-017: establece directamente la cantidad de un
// lote (usado en ajustes manuales, donde se reemplaza el valor completo).
func (r *batchRepository) UpdateQuantity(batchID uuid.UUID, newQty int) error {
	return r.db.Model(&domain.Batch{}).Where("id = ?", batchID).
		UpdateColumn("quantity", newQty).Error
}

// FindAvailableByMedicineFEFO cubre RF-025: lotes con existencias
// disponibles ordenados por fecha de vencimiento ascendente para
// la estrategia FEFO (First Expired, First Out).
func (r *batchRepository) FindAvailableByMedicineFEFO(medicineID uuid.UUID) ([]domain.Batch, error) {
	var batches []domain.Batch
	err := r.db.Where("medicine_id = ? AND quantity > 0 AND expiration_date > ?", medicineID, time.Now()).
		Order("expiration_date ASC").Find(&batches).Error
	return batches, err
}

// FindEarliestExpiration cubre RF-028: retorna el lote con vencimiento
// mas proximo para un medicamento dado (solo lotes con existencias).
func (r *batchRepository) FindEarliestExpiration(medicineID uuid.UUID) (*domain.Batch, error) {
	var batch domain.Batch
	err := r.db.Where("medicine_id = ? AND quantity > 0", medicineID).
		Order("expiration_date ASC").First(&batch).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// CountAvailableByMedicine cubre RF-028: cuenta el numero de lotes con
// existencias disponibles para un medicamento.
func (r *batchRepository) CountAvailableByMedicine(medicineID uuid.UUID) (int, error) {
	var count int64
	err := r.db.Model(&domain.Batch{}).
		Where("medicine_id = ? AND quantity > 0", medicineID).
		Count(&count).Error
	return int(count), err
}

// DecreaseQuantity cubre RF-025: resta unidades de un lote especifico
// durante el proceso FEFO de descuento de ventas.
func (r *batchRepository) DecreaseQuantity(batchID uuid.UUID, qty int) error {
	return r.db.Model(&domain.Batch{}).Where("id = ?", batchID).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", qty)).Error
}
