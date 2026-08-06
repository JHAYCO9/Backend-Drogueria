package domain

import (
	"time"

	"github.com/google/uuid"
)

// BotAuthorizedUser cubre RF-070: usuarios con acceso al bot interno.
type BotAuthorizedUser struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"uniqueIndex;not null"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
}

func (BotAuthorizedUser) TableName() string { return "bot_authorized_users" }

// BotQueryLog cubre RF-078: historial de comandos ejecutados.
type BotQueryLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"not null"`
	Command   string    `gorm:"not null"`
	Result    string    `gorm:"not null"`
	CreatedAt time.Time
}

func (BotQueryLog) TableName() string { return "bot_query_log" }

// BotConfig cubre RF-082: configuración global del bot (fila única).
type BotConfig struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	IsEnabled      bool      `gorm:"default:true"`
	WelcomeMessage string
}

func (BotConfig) TableName() string { return "bot_config" }
