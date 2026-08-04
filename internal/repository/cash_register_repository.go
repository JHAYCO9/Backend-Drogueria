package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type CashRegisterRepository interface {
	Create(cr *domain.CashRegister) error
	CreateTx(tx *gorm.DB, cr *domain.CashRegister) error // RNF-012: apertura atómica
	FindOpen() (*domain.CashRegister, error)
	FindOpenByUser(userID uuid.UUID) (*domain.CashRegister, error) // RNF-008: verifica por usuario
	FindByID(id uuid.UUID) (*domain.CashRegister, error)
	FindClosed(from, to *time.Time, userID *uuid.UUID) ([]domain.CashRegister, error) // RF-056
	Update(cr *domain.CashRegister) error
	UpdateTx(tx *gorm.DB, cr *domain.CashRegister) error // RNF-012: cierre atómico
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

// CreateTx crea una sesión de caja dentro de una transacción activa (RNF-012).
func (r *cashRegisterRepository) CreateTx(tx *gorm.DB, cr *domain.CashRegister) error {
	return tx.Create(cr).Error
}

// FindOpen busca la caja actualmente abierta (cualquier usuario).
// Esta restricción es GLOBAL: solo puede haber una caja abierta en toda la droguería.
func (r *cashRegisterRepository) FindOpen() (*domain.CashRegister, error) {
	var cr domain.CashRegister
	err := r.db.Where("status = ?", "abierta").Order("opened_at desc").First(&cr).Error
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

// FindOpenByUser busca si el usuario indicado ya tiene una caja abierta (RNF-008).
// En este sistema la restricción es global (solo una caja a la vez), pero este método
// permite también detectar si el mismo usuario intenta abrir una segunda caja.
func (r *cashRegisterRepository) FindOpenByUser(userID uuid.UUID) (*domain.CashRegister, error) {
	var cr domain.CashRegister
	err := r.db.Where("status = ? AND opened_by = ?", "abierta", userID).
		Order("opened_at desc").First(&cr).Error
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

// UpdateTx guarda cambios en una caja dentro de una transacción activa (RNF-012).
func (r *cashRegisterRepository) UpdateTx(tx *gorm.DB, cr *domain.CashRegister) error {
	return tx.Save(cr).Error
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

// FindClosed cubre RF-056: consultar historial de cierres con filtros por fecha y usuario.
func (r *cashRegisterRepository) FindClosed(from, to *time.Time, userID *uuid.UUID) ([]domain.CashRegister, error) {
	var registers []domain.CashRegister
	query := r.db.Where("status = ?", "cerrada")
	if from != nil {
		query = query.Where("closed_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("closed_at <= ?", *to)
	}
	if userID != nil {
		query = query.Where("closed_by = ?", *userID)
	}
	err := query.Order("closed_at desc").Find(&registers).Error
	return registers, err
}
