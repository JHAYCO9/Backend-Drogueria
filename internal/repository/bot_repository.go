package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type BotRepository interface {
	// RF-070
	GrantAccess(userID uuid.UUID) error
	RevokeAccess(id uuid.UUID) error
	ListAuthorized() ([]domain.BotAuthorizedUser, error)
	IsAuthorized(userID uuid.UUID) (bool, error)

	// RF-078
	LogQuery(log *domain.BotQueryLog) error
	ListQueryLogs(limit int) ([]domain.BotQueryLog, error)

	// RF-082
	GetConfig() (*domain.BotConfig, error)
	UpdateConfig(cfg *domain.BotConfig) error
}

type botRepository struct {
	db *gorm.DB
}

func NewBotRepository(db *gorm.DB) BotRepository {
	return &botRepository{db: db}
}

func (r *botRepository) GrantAccess(userID uuid.UUID) error {
	entry := &domain.BotAuthorizedUser{ID: uuid.New(), UserID: userID, IsActive: true}
	return r.db.Create(entry).Error
}

func (r *botRepository) RevokeAccess(id uuid.UUID) error {
	return r.db.Model(&domain.BotAuthorizedUser{}).Where("id = ?", id).
		UpdateColumn("is_active", false).Error
}

func (r *botRepository) ListAuthorized() ([]domain.BotAuthorizedUser, error) {
	var list []domain.BotAuthorizedUser
	err := r.db.Where("is_active = ?", true).Find(&list).Error
	return list, err
}

func (r *botRepository) IsAuthorized(userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.BotAuthorizedUser{}).
		Where("user_id = ? AND is_active = ?", userID, true).Count(&count).Error
	return count > 0, err
}

func (r *botRepository) LogQuery(log *domain.BotQueryLog) error {
	return r.db.Create(log).Error
}

func (r *botRepository) ListQueryLogs(limit int) ([]domain.BotQueryLog, error) {
	var logs []domain.BotQueryLog
	err := r.db.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}

func (r *botRepository) GetConfig() (*domain.BotConfig, error) {
	var cfg domain.BotConfig
	err := r.db.First(&cfg).Error
	return &cfg, err
}

func (r *botRepository) UpdateConfig(cfg *domain.BotConfig) error {
	return r.db.Model(&domain.BotConfig{}).Where("id = ?", cfg.ID).
		Updates(map[string]interface{}{"is_enabled": cfg.IsEnabled, "welcome_message": cfg.WelcomeMessage}).Error
}
