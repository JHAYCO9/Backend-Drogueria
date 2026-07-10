package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserAuditLog struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID
	Action    string // "login", "logout", "user_update", etc.
	Detail    string
	IPAddress string
	CreatedAt time.Time
}

func (UserAuditLog) TableName() string { return "user_audit_log" }