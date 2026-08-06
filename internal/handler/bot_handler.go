package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type BotHandler struct {
	botService service.BotService
	userRepo   repository.UserRepository
}

// Se simplifica el tipo de userRepo abajo con la interfaz real.
type BotHandlerDeps struct {
	BotService service.BotService
}

func NewBotHandler(botService service.BotService, userRepo repository.UserRepository) *BotHandler {
	return &BotHandler{botService: botService, userRepo: userRepo}
}

// ExecuteCommand cubre RF-074/075/076/077.
func (h *BotHandler) ExecuteCommand(c *gin.Context) {
	var req dto.BotQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el comando es obligatorio"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	result, err := h.botService.ExecuteCommand(userID, req.Command)
	if err != nil {
		switch err {
		case service.ErrBotDisabled:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		case service.ErrBotNotAuthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo procesar el comando"})
		}
		return
	}

	c.JSON(http.StatusOK, dto.BotQueryResponse{Command: req.Command, Result: result})
}

// GetPendingAlerts cubre RF-071/072/073.
func (h *BotHandler) GetPendingAlerts(c *gin.Context) {
	alerts, err := h.botService.GetPendingAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las alertas"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alerts": alerts})
}

// GrantAccess cubre RF-070.
func (h *BotHandler) GrantAccess(c *gin.Context) {
	var req dto.GrantBotAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id es obligatorio"})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id inválido"})
		return
	}

	if err := h.botService.GrantAccess(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo otorgar el acceso"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "acceso otorgado correctamente"})
}

func (h *BotHandler) RevokeAccess(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	if err := h.botService.RevokeAccess(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo revocar el acceso"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "acceso revocado correctamente"})
}

func (h *BotHandler) ListAuthorized(c *gin.Context) {
	list, err := h.botService.ListAuthorized()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar la lista"})
		return
	}

	response := make([]dto.BotAuthorizedUserResponse, 0, len(list))
	for _, item := range list {
		username, fullName := "", ""
		if u, err := h.userRepo.FindByID(item.UserID); err == nil {
			username = u.Username
			fullName = u.FullName
		}
		response = append(response, dto.BotAuthorizedUserResponse{
			ID: item.ID.String(), UserID: item.UserID.String(),
			Username: username, FullName: fullName,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, response)
}

// ListQueryLogs cubre RF-078.
func (h *BotHandler) ListQueryLogs(c *gin.Context) {
	logs, err := h.botService.ListQueryLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el historial"})
		return
	}

	response := make([]dto.BotQueryLogResponse, 0, len(logs))
	for _, l := range logs {
		username := ""
		if u, err := h.userRepo.FindByID(l.UserID); err == nil {
			username = u.Username
		}
		response = append(response, dto.BotQueryLogResponse{
			Username: username, Command: l.Command, Result: l.Result,
			CreatedAt: l.CreatedAt.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, response)
}

// GetConfig / UpdateConfig cubren RF-082.
func (h *BotHandler) GetConfig(c *gin.Context) {
	cfg, err := h.botService.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar la configuración"})
		return
	}
	c.JSON(http.StatusOK, dto.BotConfigResponse{IsEnabled: cfg.IsEnabled, WelcomeMessage: cfg.WelcomeMessage})
}

func (h *BotHandler) UpdateConfig(c *gin.Context) {
	var req dto.UpdateBotConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	if err := h.botService.UpdateConfig(req.IsEnabled, req.WelcomeMessage); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la configuración"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "configuración actualizada correctamente"})
}
