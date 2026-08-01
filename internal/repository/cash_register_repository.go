package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type CashRegisterRepository interface {
	Create(cr *domain.CashRegister) error
	FindOpen() (*domain.CashRegister, error)
	FindByID(id uuid.UUID) (*domain.CashRegister, error)
	Update(cr *domain.CashRegister) error
	CountClosedUpTo(closedAt time.Time) (int64, error)
}

type cashRegisterRepository struct {
	db *gorm.DB
}

func NewCashRegisterRepository(db *gorm.DB) CashRegisterRepository {
	return &cashRegisterRepository{db: db}
}

func (r *cashRegisterRepository) Create(cr *domain.CashRegister) error {
	return r.db.Create(cr).Error
}

func (r *cashRegisterRepository) FindOpen() (*domain.CashRegister, error) {
	var cr domain.CashRegister
	err := r.db.Where("status = ?", "abierta").Order("opened_at desc").First(&cr).Error
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *cashRegisterRepository) FindByID(id uuid.UUID) (*domain.CashRegister, error) {
	var cr domain.CashRegister
	if err := r.db.First(&cr, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &cr, nil
}

func (r *cashRegisterRepository) Update(cr *domain.CashRegister) error {
	return r.db.Save(cr).Error
}

// CountClosedUpTo cubre el "Z Nro." de RF-055: cuenta cuántas cajas se
// han cerrado hasta (e incluyendo) la fecha indicada, para numerar el
// reporte Z de forma secuencial.
func (r *cashRegisterRepository) CountClosedUpTo(closedAt time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&domain.CashRegister{}).
		Where("status = ? AND closed_at <= ?", "cerrada", closedAt).Count(&count).Error
	return count, err
}
