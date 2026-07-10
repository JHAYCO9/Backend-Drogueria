package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type AuditRepository interface {
	Log(userID uuid.UUID, action, detail, ipAddress string) error
	FindAll() ([]domain.UserAuditLog, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Log(userID uuid.UUID, action, detail, ipAddress string) error {
	entry := domain.UserAuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		Detail:    detail,
		IPAddress: ipAddress,
	}
	return r.db.Create(&entry).Error
}

// FindAll devuelve todos los registros de auditoría ordenados del más
// reciente al más antiguo (RF-010).
func (r *auditRepository) FindAll() ([]domain.UserAuditLog, error) {
	var logs []domain.UserAuditLog
	err := r.db.Order("created_at DESC").Find(&logs).Error
	return logs, err
}