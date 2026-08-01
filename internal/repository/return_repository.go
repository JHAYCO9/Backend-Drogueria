package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type ReturnRepository interface {
	CreateTx(tx *gorm.DB, r *domain.Return) error
	FindAll() ([]domain.Return, error) // RF-093
	SumReturnedForSaleItem(saleItemID uuid.UUID) (int, error)
	ExistsForSale(saleID uuid.UUID) (bool, error)
}

type returnRepository struct {
	db *gorm.DB
}

func NewReturnRepository(db *gorm.DB) ReturnRepository {
	return &returnRepository{db: db}
}

func (r *returnRepository) CreateTx(tx *gorm.DB, ret *domain.Return) error {
	return tx.Create(ret).Error
}

// FindAll cubre RF-093: historial completo de devoluciones.
func (r *returnRepository) FindAll() ([]domain.Return, error) {
	var returns []domain.Return
	err := r.db.Order("created_at desc").Find(&returns).Error
	return returns, err
}

// SumReturnedForSaleItem evita que se devuelva más de lo que realmente
// se vendió, sumando todo lo ya devuelto previamente para ese ítem.
func (r *returnRepository) SumReturnedForSaleItem(saleItemID uuid.UUID) (int, error) {
	var total int
	err := r.db.Model(&domain.Return{}).Where("sale_item_id = ?", saleItemID).
		Select("COALESCE(SUM(quantity), 0)").Scan(&total).Error
	return total, err
}
func (r *returnRepository) ExistsForSale(saleID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Return{}).Where("sale_id = ?", saleID).Count(&count).Error
	return count > 0, err
}
