package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type CashMovementFilter struct {
	UserID *uuid.UUID
	From   *time.Time
	To     *time.Time
}

type CashMovementRepository interface {
	Create(m *domain.CashMovement) error
	FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error)
	FindFiltered(filter CashMovementFilter) ([]domain.CashMovement, error) // RF-051
	FindRecentByTypes(types []string, limit int) ([]domain.CashMovement, error)
}

type cashMovementRepository struct {
	db *gorm.DB
}

func NewCashMovementRepository(db *gorm.DB) CashMovementRepository {
	return &cashMovementRepository{db: db}
}

func (r *cashMovementRepository) Create(m *domain.CashMovement) error {
	return r.db.Create(m).Error
}

func (r *cashMovementRepository) FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error) {
	var movements []domain.CashMovement
	err := r.db.Where("cash_register_id = ?", cashRegisterID).Order("created_at").Find(&movements).Error
	return movements, err
}

// FindRecentByTypes cubre RF-060: últimos movimientos de caja de ciertos
// tipos (ej. apertura, cierre), para el feed de actividad reciente.
func (r *cashMovementRepository) FindRecentByTypes(types []string, limit int) ([]domain.CashMovement, error) {
	var movements []domain.CashMovement
	err := r.db.Where("movement_type IN ?", types).
		Order("created_at DESC").Limit(limit).Find(&movements).Error
	return movements, err
}

// FindFiltered cubre RF-051: filtro por fecha y usuario.
func (r *cashMovementRepository) FindFiltered(filter CashMovementFilter) ([]domain.CashMovement, error) {
	query := r.db.Model(&domain.CashMovement{})

	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", *filter.To)
	}

	var movements []domain.CashMovement
	err := query.Order("created_at desc").Find(&movements).Error
	return movements, err
}
