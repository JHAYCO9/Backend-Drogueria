package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "usuario y contraseña son obligatorios"})
		return
	}

	token, roleName, err := h.authService.Login(req.Username, req.Password, c.ClientIP())
	if err != nil {
		switch err {
		case service.ErrUserLocked:
			c.JSON(http.StatusForbidden, gin.H{"error": "cuenta bloqueada temporalmente por múltiples intentos fallidos, intenta de nuevo en unos minutos"})
		case service.ErrUserInactive:
			c.JSON(http.StatusForbidden, gin.H{"error": "usuario desactivado"})
		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario o contraseña incorrectos"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.LoginResponse{Token: token, RoleName: roleName})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")
	if err := h.authService.Logout(userID.(string), c.ClientIP()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar sesión"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sesión finalizada correctamente"})
}
