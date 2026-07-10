package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrInvalidCredentials = errors.New("usuario o contraseña incorrectos")
var ErrUserLocked = errors.New("cuenta bloqueada temporalmente por múltiples intentos fallidos")
var ErrUserInactive = errors.New("usuario desactivado")

const (
	maxFailedAttempts = 5
	lockDuration      = 15 * time.Minute
)

type AuthService interface {
	Login(username, password, ip string) (token, roleName string, err error)
	Logout(userIDStr, ip string) error
}

type authService struct {
	userRepo  repository.UserRepository
	roleRepo  repository.RoleRepository
	auditRepo repository.AuditRepository
	jwtSecret string
}

func NewAuthService(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auditRepo repository.AuditRepository, jwtSecret string) AuthService {
	return &authService{userRepo: userRepo, roleRepo: roleRepo, auditRepo: auditRepo, jwtSecret: jwtSecret}
}

func (s *authService) Login(username, password, ip string) (string, string, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}

	if !user.IsActive {
		return "", "", ErrUserInactive
	}

	if user.LockedUntil != nil && user.LockedUntil.After(time.Now()) {
		_ = s.auditRepo.Log(user.ID, "login_blocked", "intento de acceso con cuenta bloqueada", ip)
		return "", "", ErrUserLocked
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		locked, incErr := s.userRepo.IncrementFailedLogin(user.ID, maxFailedAttempts, lockDuration)
		if incErr != nil {
			return "", "", ErrInvalidCredentials
		}

		if locked {
			_ = s.auditRepo.Log(user.ID, "account_locked", "cuenta bloqueada por 5 intentos fallidos consecutivos", ip)
			return "", "", ErrUserLocked
		}

		return "", "", ErrInvalidCredentials
	}

	_ = s.userRepo.ResetFailedLogin(user.ID)

	now := time.Now()
	_ = s.userRepo.UpdateLastLogin(user.ID, now)
	_ = s.auditRepo.Log(user.ID, "login", "inicio de sesión exitoso", ip)

	roleName := ""
	if role, err := s.roleRepo.FindByID(user.RoleID); err == nil && role != nil {
		roleName = role.Name
	}

	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"role_id": user.RoleID.String(),
		"exp":     now.Add(8 * time.Hour).Unix(),
		"iat":     now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return signed, roleName, nil
}

func (s *authService) Logout(userIDStr, ip string) error {
	uid, err := parseUUID(userIDStr)
	if err != nil {
		return err
	}
	return s.auditRepo.Log(uid, "logout", "cierre de sesión", ip)
}
