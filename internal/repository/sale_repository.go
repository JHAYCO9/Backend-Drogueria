package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

// SaleFilterParams agrupa los filtros opcionales para ListSalesFiltered.
type SaleFilterParams struct {
	StartDate  *time.Time
	EndDate    *time.Time
	CustomerID *uuid.UUID
	UserID     *uuid.UUID
	Status     string
	Page       int // 1-based
	PageSize   int
}

type SaleRepository interface {
	Create(sale *domain.Sale) error
	FindByID(id uuid.UUID) (*domain.Sale, error)
	FindByIDTx(tx *gorm.DB, id uuid.UUID) (*domain.Sale, error)
	Update(sale *domain.Sale) error
	UpdateTx(tx *gorm.DB, sale *domain.Sale) error
	AddItem(item *domain.SaleItem) error
	UpdateItem(item *domain.SaleItem) error
	RemoveItem(itemID uuid.UUID, saleID uuid.UUID) error
	FindItems(saleID uuid.UUID) ([]domain.SaleItem, error)
	FindItemsTx(tx *gorm.DB, saleID uuid.UUID) ([]domain.SaleItem, error)
	FindItemByID(itemID uuid.UUID) (*domain.SaleItem, error)
	FindByDateRange(from, to time.Time) ([]domain.Sale, error)
	FindAll() ([]domain.Sale, error)
	FindInProgress(cashRegisterID uuid.UUID) (*domain.Sale, error)
	FindWithFilters(params SaleFilterParams) ([]domain.Sale, int64, error)
	FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.Sale, error)
	FindTodayFinalized(from, to time.Time) ([]domain.Sale, error)
	// RF-019/RF-017: sumas para el arqueo de caja.
	SumCashSalesByCashRegister(cashRegisterID uuid.UUID) (float64, error)
	SumOtherSalesByCashRegister(cashRegisterID uuid.UUID) (float64, error)
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

func (r *saleRepository) FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.Sale, error) {
	var sales []domain.Sale
	err := r.db.Where("cash_register_id = ?", cashRegisterID).Find(&sales).Error
	return sales, err
}

func (r *saleRepository) Update(sale *domain.Sale) error {
	return r.db.Save(sale).Error
}

func (r *saleRepository) AddItem(item *domain.SaleItem) error {
	return r.db.Create(item).Error
}

func (r *saleRepository) UpdateItem(item *domain.SaleItem) error {
	return r.db.Save(item).Error
}

// RemoveItem cubre RF-035: valida que el ítem pertenezca a la venta
// indicada, para que un cajero no pueda borrar líneas de otra venta
// manipulando el ID directamente.
func (r *saleRepository) RemoveItem(itemID uuid.UUID, saleID uuid.UUID) error {
	return r.db.Where("id = ? AND sale_id = ?", itemID, saleID).Delete(&domain.SaleItem{}).Error
}

// FindTodayFinalized cubre RF-058: ventas finalizadas dentro del rango
// de fechas indicado (calculado por el caller con la zona horaria
// correcta, para no depender de la zona horaria del servidor de BD).
func (r *saleRepository) FindTodayFinalized(from, to time.Time) ([]domain.Sale, error) {
	var sales []domain.Sale
	err := r.db.Where("status = ? AND created_at >= ? AND created_at < ?", "finalizada", from, to).
		Find(&sales).Error
	return sales, err
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

// FindByIDTx permite buscar una venta dentro de una transacción activa.
func (r *saleRepository) FindByIDTx(tx *gorm.DB, id uuid.UUID) (*domain.Sale, error) {
	var sale domain.Sale
	if err := tx.First(&sale, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &sale, nil
}

// UpdateTx guarda cambios en una venta dentro de una transacción activa.
func (r *saleRepository) UpdateTx(tx *gorm.DB, sale *domain.Sale) error {
	return tx.Save(sale).Error
}

// FindItemsTx devuelve los ítems de una venta dentro de una transacción activa.
func (r *saleRepository) FindItemsTx(tx *gorm.DB, saleID uuid.UUID) ([]domain.SaleItem, error) {
	var items []domain.SaleItem
	err := tx.Where("sale_id = ?", saleID).Order("created_at").Find(&items).Error
	return items, err
}

// FindWithFilters cubre RF-022: listado de ventas paginado con filtros
// opcionales por rango de fecha, cliente, usuario y estado.
func (r *saleRepository) FindWithFilters(params SaleFilterParams) ([]domain.Sale, int64, error) {
	var sales []domain.Sale
	var total int64

	query := r.db.Model(&domain.Sale{})

	if params.StartDate != nil {
		query = query.Where("created_at >= ?", params.StartDate)
	}
	if params.EndDate != nil {
		query = query.Where("created_at <= ?", params.EndDate)
	}
	if params.CustomerID != nil {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.UserID != nil {
		query = query.Where("user_id = ?", params.UserID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := params.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * pageSize

	err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&sales).Error
	return sales, total, err
}

// SumCashSalesByCashRegister calcula el total de ventas en EFECTIVO dentro
// de una sesión de caja (status='finalizada' y payment_method='efectivo').
// Usado en la fórmula del arqueo: Monto_Esperado = Inicial + Ventas_Efectivo + Ingresos - Egresos.
func (r *saleRepository) SumCashSalesByCashRegister(cashRegisterID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.Model(&domain.Sale{}).
		Where("cash_register_id = ? AND status = ? AND payment_method = ?",
			cashRegisterID, "finalizada", "efectivo").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&total).Error
	return total, err
}

// SumOtherSalesByCashRegister calcula el total de ventas POR OTROS MEDIOS
// (Nequi, transferencia, etc.) dentro de una sesión de caja.
// Este valor es informativo: ese dinero no entra físico a la caja.
func (r *saleRepository) SumOtherSalesByCashRegister(cashRegisterID uuid.UUID) (float64, error) {
	var total float64
	err := r.db.Model(&domain.Sale{}).
		Where("cash_register_id = ? AND status = ? AND payment_method != ?",
			cashRegisterID, "finalizada", "efectivo").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&total).Error
	return total, err
}
