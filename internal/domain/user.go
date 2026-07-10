package domain

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	FullName         string
	DocumentID       string `gorm:"uniqueIndex"`
	Username         string `gorm:"uniqueIndex"`
	Email            *string
	PasswordHash     string
	RoleID           uuid.UUID
	IsActive         bool
	FailedLoginCount int
	LockedUntil      *time.Time
	LastLoginAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (User) TableName() string { return "users" }
