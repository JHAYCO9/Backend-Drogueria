package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type UserRepository interface {
	FindByUsername(username string) (*domain.User, error)
	FindByID(id uuid.UUID) (*domain.User, error)
	FindAll() ([]domain.User, error)
	IncrementFailedLogin(userID uuid.UUID, maxAttempts int, lockDuration time.Duration) (bool, error)
	ResetFailedLogin(userID uuid.UUID) error
	UpdateLastLogin(userID uuid.UUID, t time.Time) error
	Update(user *domain.User) error
	Create(user *domain.User) error
	ExistsByUsername(username string) (bool, error)
	Deactivate(userID uuid.UUID) error
	Activate(userID uuid.UUID) error
	UpdatePassword(userID uuid.UUID, newHashedPassword string) error
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByUsername(username string) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) FindByID(id uuid.UUID) (*domain.User, error) {
	var user domain.User
	if err := r.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// IncrementFailedLogin suma un intento fallido y, si al sumar alcanza el
// umbral, bloquea la cuenta por lockDuration a partir de ahora (RF-009).
// Devuelve true si esta llamada fue la que activó el bloqueo.
func (r *userRepository) IncrementFailedLogin(userID uuid.UUID, maxAttempts int, lockDuration time.Duration) (locked bool, err error) {
	var user domain.User
	if err := r.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, err
	}

	newCount := user.FailedLoginCount + 1
	updates := map[string]interface{}{
		"failed_login_count": newCount,
	}

	locked = newCount >= maxAttempts
	if locked {
		lockedUntil := time.Now().Add(lockDuration)
		updates["locked_until"] = lockedUntil
	}

	err = r.db.Model(&domain.User{}).Where("id = ?", userID).Updates(updates).Error
	return locked, err
}

func (r *userRepository) FindAll() ([]domain.User, error) {
	var users []domain.User
	err := r.db.Order("full_name").Find(&users).Error
	return users, err
}

func (r *userRepository) ResetFailedLogin(userID uuid.UUID) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).
		Updates(map[string]interface{}{"failed_login_count": 0, "locked_until": nil}).Error
}

func (r *userRepository) UpdateLastLogin(userID uuid.UUID, t time.Time) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).
		UpdateColumn("last_login_at", t).Error
}

// Update sobrescribe únicamente los campos editables por RF-004
// (nombre, usuario, rol, estado) — nunca la contraseña desde aquí.
func (r *userRepository) Update(user *domain.User) error {
	return r.db.Model(&domain.User{}).Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"full_name": user.FullName,
			"username":  user.Username,
			"role_id":   user.RoleID,
			"is_active": user.IsActive,
		}).Error
}

// Create inserta un nuevo usuario. Devuelve error si ya existe
// (constraint unique de username o document_id), que el service
// interpreta para dar el mensaje correcto.
func (r *userRepository) Create(user *domain.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// Deactivate pone is_active = false sin eliminar el registro (RF-005).
// Se usa UpdateColumn para evitar que GORM dispare callbacks innecesarios.
func (r *userRepository) Deactivate(userID uuid.UUID) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).
		UpdateColumn("is_active", false).Error
}

// Activate vuelve a habilitar un usuario previamente desactivado.
func (r *userRepository) Activate(userID uuid.UUID) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).
		UpdateColumn("is_active", true).Error
}

// UpdatePassword modifica únicamente el hash de la contraseña (RF-006).
func (r *userRepository) UpdatePassword(userID uuid.UUID, newHashedPassword string) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).
		UpdateColumn("password_hash", newHashedPassword).Error
}
