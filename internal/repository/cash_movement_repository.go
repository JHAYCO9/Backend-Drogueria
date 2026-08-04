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
	CreateTx(tx *gorm.DB, m *domain.CashMovement) error // RNF-012: escritura atómica
	FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error)
	FindManualByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error) // RF-019: solo manuales
	FindFiltered(filter CashMovementFilter) ([]domain.CashMovement, error)            // RF-051
	FindRecentByTypes(types []string, limit int) ([]domain.CashMovement, error)
	// RF-019: suma de montos por tipo de movimiento y sesión para el arqueo.
	SumByTypeAndSession(cashRegisterID uuid.UUID, movType string) (float64, error)
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

// CreateTx inserta un movimiento dentro de una transacción activa (RNF-012).
func (r *cashMovementRepository) CreateTx(tx *gorm.DB, m *domain.CashMovement) error {
	return tx.Create(m).Error
}

func (r *cashMovementRepository) FindByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error) {
	var movements []domain.CashMovement
	err := r.db.Where("cash_register_id = ?", cashRegisterID).Order("created_at").Find(&movements).Error
	return movements, err
}

// FindManualByCashRegister retorna solo los movimientos manuales (RF-018/RF-019):
// ingresos y egresos registrados explícitamente por el cajero durante la sesión.
func (r *cashMovementRepository) FindManualByCashRegister(cashRegisterID uuid.UUID) ([]domain.CashMovement, error) {
	var movements []domain.CashMovement
	err := r.db.Where("cash_register_id = ? AND movement_category = ?", cashRegisterID, "manual").
		Order("created_at").Find(&movements).Error
	return movements, err
}

// SumByTypeAndSession calcula el total acumulado de movimientos de un tipo
// específico para una sesión de caja. Usado en el arqueo (RF-019/RF-017).
func (r *cashMovementRepository) SumByTypeAndSession(cashRegisterID uuid.UUID, movType string) (float64, error) {
	var total float64
	query := r.db.Model(&domain.CashMovement{}).Where("cash_register_id = ?", cashRegisterID)

	switch movType {
	case "ingreso_manual", "ingreso":
		query = query.Where(
			"movement_category = 'manual' AND (movement_type IN ('ingreso_manual', 'ingreso') OR (movement_type = 'ajuste' AND amount > 0))",
		).Select("COALESCE(SUM(amount), 0)")
	case "egreso_manual", "egreso":
		query = query.Where(
			"movement_category = 'manual' AND (movement_type IN ('egreso_manual', 'egreso') OR (movement_type = 'ajuste' AND amount < 0))",
		).Select("COALESCE(SUM(ABS(amount)), 0)")
	default:
		query = query.Where("movement_type = ?", movType).Select("COALESCE(SUM(amount), 0)")
	}

	err := query.Scan(&total).Error
	return total, err
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
