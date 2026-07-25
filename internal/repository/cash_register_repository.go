package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type CashRegisterRepository interface {
	FindOpen() (*domain.CashRegister, error)
}

type cashRegisterRepository struct {
	db *gorm.DB
}

func NewCashRegisterRepository(db *gorm.DB) CashRegisterRepository {
	return &cashRegisterRepository{db: db}
}

// FindOpen es la base de RF-032: si no hay ninguna fila con status
// "abierta", no existe jornada activa.
func (r *cashRegisterRepository) FindOpen() (*domain.CashRegister, error) {
	var cr domain.CashRegister
	err := r.db.Where("status = ?", "abierta").Order("opened_at desc").First(&cr).Error
	if err != nil {
		return nil, err
	}
	return &cr, nil
}

var _ = uuid.Nil // evita import sin uso si luego quitas algo
