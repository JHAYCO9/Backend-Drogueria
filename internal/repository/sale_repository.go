package repository

import (
	"time"

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
	FindByDateRange(from, to time.Time) ([]domain.Sale, error)
	FindAll() ([]domain.Sale, error)
	FindInProgress(cashRegisterID uuid.UUID) (*domain.Sale, error)
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

// FindInProgress busca si ya existe una venta sin finalizar para esta
// caja — evita crear ventas huérfanas cada vez que el cajero entra al
// módulo sin haber completado la anterior.
func (r *saleRepository) FindInProgress(cashRegisterID uuid.UUID) (*domain.Sale, error) {
	var sale domain.Sale
	err := r.db.Where("cash_register_id = ? AND status = ?", cashRegisterID, "en_progreso").
		Order("created_at desc").First(&sale).Error
	if err != nil {
		return nil, err
	}
	return &sale, nil
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

// FindByDateRange cubre RF-047: solo ventas finalizadas dentro del rango
// (las canceladas o en progreso no cuentan como "comportamiento comercial").
func (r *saleRepository) FindByDateRange(from, to time.Time) ([]domain.Sale, error) {
	var sales []domain.Sale
	err := r.db.Where("status = ? AND created_at >= ? AND created_at <= ?", "finalizada", from, to).
		Order("created_at desc").Find(&sales).Error
	return sales, err
}

// FindAll cubre RF-044: historial completo, más reciente primero.
func (r *saleRepository) FindAll() ([]domain.Sale, error) {
	var sales []domain.Sale
	err := r.db.Order("created_at desc").Find(&sales).Error
	return sales, err
}
