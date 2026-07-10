package service

import (
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrUserNotFound = errors.New("usuario no encontrado")
var ErrUsernameTaken = errors.New("ya existe un usuario con ese nombre de usuario")
var ErrRoleNotFound = errors.New("el rol indicado no existe")
var ErrPasswordMismatch = errors.New("la nueva contraseña y la confirmación no coinciden")
var ErrCurrentPasswordWrong = errors.New("la contraseña actual es incorrecta")

type UserListItem struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Username string `json:"username"`
	RoleID   string `json:"role_id"`
	IsActive bool   `json:"is_active"`
}

type RegisterUserInput struct {
	FullName          string
	DocumentID        string
	Email             *string
	Username          string
	TemporaryPassword string
	RoleID            uuid.UUID
}

type UpdateUserInput struct {
	FullName string
	Username string
	RoleID   uuid.UUID
	IsActive bool
}

type UserService interface {
	RegisterUser(input RegisterUserInput, actorUserID uuid.UUID, ip string) (*domain.User, error)
	UpdateUser(targetUserID uuid.UUID, input UpdateUserInput, actorUserID uuid.UUID, ip string) error
	ListUsers() ([]UserListItem, error)
	DeactivateUser(targetUserID, actorUserID uuid.UUID, ip string) error
	ActivateUser(targetUserID, actorUserID uuid.UUID, ip string) error
	ChangePassword(userID uuid.UUID, currentPassword, newPassword, confirmPassword, ip string) error
}

type userService struct {
	userRepo  repository.UserRepository
	roleRepo  repository.RoleRepository
	auditRepo repository.AuditRepository
}

func NewUserService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auditRepo repository.AuditRepository) UserService {
	return &userService{userRepo: userRepo, roleRepo: roleRepo, auditRepo: auditRepo}
}

func (s *userService) ListUsers() ([]UserListItem, error) {
	users, err := s.userRepo.FindAll()
	if err != nil {
		return nil, err
	}
	result := make([]UserListItem, 0, len(users))
	for _, u := range users {
		result = append(result, UserListItem{
			ID: u.ID.String(), FullName: u.FullName, Username: u.Username,
			RoleID: u.RoleID.String(), IsActive: u.IsActive,
		})
	}
	return result, nil
}

func (s *userService) RegisterUser(input RegisterUserInput, actorUserID uuid.UUID, ip string) (*domain.User, error) {
	exists, err := s.userRepo.ExistsByUsername(input.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	if _, err := s.roleRepo.FindByID(input.RoleID); err != nil {
		return nil, ErrRoleNotFound
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.TemporaryPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		ID:           uuid.New(),
		FullName:     input.FullName,
		DocumentID:   input.DocumentID,
		Email:        normalizeEmail(input.Email),
		Username:     input.Username,
		PasswordHash: string(hash),
		RoleID:       input.RoleID,
		IsActive:     true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	detail := "registro de nuevo usuario " + user.Username
	_ = s.auditRepo.Log(actorUserID, "user_create", detail, ip)

	return user, nil
}

func normalizeEmail(email *string) *string {
	if email == nil || *email == "" {
		return nil
	}
	return email
}

func (s *userService) UpdateUser(targetUserID uuid.UUID, input UpdateUserInput, actorUserID uuid.UUID, ip string) error {
	user, err := s.userRepo.FindByID(targetUserID)
	if err != nil {
		return ErrUserNotFound
	}
	user.FullName = input.FullName
	user.Username = input.Username
	user.RoleID = input.RoleID
	user.IsActive = input.IsActive

	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	detail := "actualización de datos de usuario " + user.Username
	return s.auditRepo.Log(actorUserID, "user_update", detail, ip)
}

// DeactivateUser cambia is_active a false sin eliminar el registro
// (soft delete de estado — RF-005). Registra la acción en auditoría.
func (s *userService) DeactivateUser(targetUserID, actorUserID uuid.UUID, ip string) error {
	// Verificar que el usuario existe antes de desactivarlo
	user, err := s.userRepo.FindByID(targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	// Actualizar estado a inactivo en la BD
	if err := s.userRepo.Deactivate(targetUserID); err != nil {
		return err
	}

	detail := "desactivación de usuario " + user.Username
	_ = s.auditRepo.Log(actorUserID, "user_deactivate", detail, ip)
	return nil
}

// ActivateUser vuelve a habilitar un usuario previamente desactivado.
// Registra la acción en la auditoría.
func (s *userService) ActivateUser(targetUserID, actorUserID uuid.UUID, ip string) error {
	user, err := s.userRepo.FindByID(targetUserID)
	if err != nil {
		return ErrUserNotFound
	}

	if err := s.userRepo.Activate(targetUserID); err != nil {
		return err
	}

	detail := "activación de usuario " + user.Username
	_ = s.auditRepo.Log(actorUserID, "user_activate", detail, ip)
	return nil
}

// ChangePassword valida la contraseña actual, verifica que la nueva y
// su confirmación coincidan, encripta la nueva y la guarda (RF-006).
func (s *userService) ChangePassword(userID uuid.UUID, currentPassword, newPassword, confirmPassword, ip string) error {
	// 1. Validar que la nueva contraseña y la confirmación coincidan
	if newPassword != confirmPassword {
		return ErrPasswordMismatch
	}

	// 2. Obtener el usuario de la BD
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// 3. Verificar que la contraseña actual sea correcta (bcrypt)
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)) != nil {
		return ErrCurrentPasswordWrong
	}

	// 4. Encriptar la nueva contraseña
	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 5. Guardar el nuevo hash en la BD
	if err := s.userRepo.UpdatePassword(userID, string(newHash)); err != nil {
		return err
	}

	// 6. Registrar la acción en la auditoría
	_ = s.auditRepo.Log(userID, "password_change", "cambio de contraseña exitoso", ip)
	return nil
}
