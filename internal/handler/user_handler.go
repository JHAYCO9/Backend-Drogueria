package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de usuario inválido"})
		return
	}

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_id inválido"})
		return
	}

	actorIDStr, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(actorIDStr.(string))

	err = h.userService.UpdateUser(targetID, service.UpdateUserInput{
		FullName: req.FullName,
		Username: req.Username,
		RoleID:   roleID,
		IsActive: req.IsActive,
	}, actorID, c.ClientIP())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuario actualizado correctamente"})
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	users, err := h.userService.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los usuarios"})
		return
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req dto.RegisterUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	roleID, err := uuid.Parse(req.RoleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role_id inválido"})
		return
	}

	actorIDStr, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(actorIDStr.(string))

	user, err := h.userService.RegisterUser(service.RegisterUserInput{
		FullName:          req.FullName,
		DocumentID:        req.DocumentID,
		Email:             req.Email,
		Username:          req.Username,
		TemporaryPassword: req.TemporaryPassword,
		RoleID:            roleID,
	}, actorID, c.ClientIP())

	if err != nil {
		switch err {
		case service.ErrUsernameTaken:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrRoleNotFound:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el usuario"})
		}
		return
	}

	c.JSON(http.StatusCreated, dto.RegisterUserResponse{ID: user.ID.String(), Username: user.Username})
}

// DeactivateUser desactiva un usuario sin eliminarlo (RF-005).
// PATCH /api/v1/users/:id/deactivate — Solo Administrador.
func (h *UserHandler) DeactivateUser(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de usuario inválido"})
		return
	}

	actorIDStr, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(actorIDStr.(string))

	err = h.userService.DeactivateUser(targetID, actorID, c.ClientIP())
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo desactivar el usuario"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuario desactivado correctamente"})
}

// ActivateUser vuelve a habilitar un usuario previamente desactivado.
// PATCH /api/v1/users/:id/activate — Solo Administrador.
func (h *UserHandler) ActivateUser(c *gin.Context) {
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de usuario inválido"})
		return
	}

	actorIDStr, _ := c.Get("user_id")
	actorID, _ := uuid.Parse(actorIDStr.(string))

	err = h.userService.ActivateUser(targetID, actorID, c.ClientIP())
	if err != nil {
		switch err {
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo activar el usuario"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "usuario activado correctamente"})
}

// ChangePassword permite al usuario autenticado cambiar su propia
// contraseña (RF-006). PUT /api/v1/users/change-password.
func (h *UserHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	// El user_id viene del JWT — el usuario se cambia su propia contraseña
	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	err := h.userService.ChangePassword(
		userID,
		req.CurrentPassword,
		req.NewPassword,
		req.ConfirmPassword,
		c.ClientIP(),
	)

	if err != nil {
		switch err {
		case service.ErrPasswordMismatch:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case service.ErrCurrentPasswordWrong:
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		case service.ErrUserNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cambiar la contraseña"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "contraseña actualizada correctamente"})
}
