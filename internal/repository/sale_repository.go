package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type SaleRepository interface {
	Create(sale *domain.Sale) error
	FindByID(id uuid.UUID) (*domain.Sale, error)
	Update(sale *domain.Sale) error
	AddItem(item *domain.SaleItem) error
	RemoveItem(itemID uuid.UUID, saleID uuid.UUID) error
	FindItems(saleID uuid.UUID) ([]domain.SaleItem, error)
	FindItemByID(itemID uuid.UUID) (*domain.SaleItem, error)
}

type saleRepository struct {
	db *gorm.DB
}

func NewSaleRepository(db *gorm.DB) SaleRepository {
	return &saleRepository{db: db}
}

func (r *saleRepository) Create(sale *domain.Sale) error {
	return r.db.Create(sale).Error
}

func (r *saleRepository) FindByID(id uuid.UUID) (*domain.Sale, error) {
	var sale domain.Sale
	if err := r.db.First(&sale, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sale, nil
}

func (r *saleRepository) Update(sale *domain.Sale) error {
	return r.db.Save(sale).Error
}

func (r *saleRepository) AddItem(item *domain.SaleItem) error {
	return r.db.Create(item).Error
}

// RemoveItem cubre RF-035: valida que el ítem pertenezca a la venta
// indicada, para que un cajero no pueda borrar líneas de otra venta
// manipulando el ID directamente.
func (r *saleRepository) RemoveItem(itemID uuid.UUID, saleID uuid.UUID) error {
	return r.db.Where("id = ? AND sale_id = ?", itemID, saleID).Delete(&domain.SaleItem{}).Error
}

func (r *saleRepository) FindItems(saleID uuid.UUID) ([]domain.SaleItem, error) {
	var items []domain.SaleItem
	err := r.db.Where("sale_id = ?", saleID).Order("created_at").Find(&items).Error
	return items, err
}

func (r *saleRepository) FindItemByID(itemID uuid.UUID) (*domain.SaleItem, error) {
	var item domain.SaleItem
	if err := r.db.First(&item, "id = ?", itemID).Error; err != nil {
		return nil, err
	}
	return &item, nil
}
