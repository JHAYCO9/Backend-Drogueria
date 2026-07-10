package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type RoleRepository interface {
	FindByID(id uuid.UUID) (*domain.Role, error)
	FindAll() ([]domain.Role, error) // <- nuevo
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) FindByID(id uuid.UUID) (*domain.Role, error) {
	var role domain.Role
	if err := r.db.First(&role, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *roleRepository) FindAll() ([]domain.Role, error) {
	var roles []domain.Role
	err := r.db.Order("name").Find(&roles).Error
	return roles, err
}
