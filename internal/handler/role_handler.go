package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type RoleHandler struct {
	roleRepo repository.RoleRepository
}

func NewRoleHandler(roleRepo repository.RoleRepository) *RoleHandler {
	return &RoleHandler{roleRepo: roleRepo}
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	roles, err := h.roleRepo.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los roles"})
		return
	}
	c.JSON(http.StatusOK, roles)
}
