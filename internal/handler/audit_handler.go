package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

// AuditHandler expone el endpoint de consulta de auditoría (RF-010).
type AuditHandler struct {
	auditService service.AuditService
}

func NewAuditHandler(auditService service.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetAuditLogs devuelve el listado completo de registros de auditoría
// en formato JSON. Solo accesible para el rol Administrador (protegido
// desde el router con RequireRole).
func (h *AuditHandler) GetAuditLogs(c *gin.Context) {
	logs, err := h.auditService.GetAllAuditLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los registros de auditoría"})
		return
	}
	c.JSON(http.StatusOK, logs)
}
