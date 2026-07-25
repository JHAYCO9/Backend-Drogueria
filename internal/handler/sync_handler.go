package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type SyncHandler struct {
	syncRepo repository.SyncRepository
}

func NewSyncHandler(syncRepo repository.SyncRepository) *SyncHandler {
	return &SyncHandler{syncRepo: syncRepo}
}

// PushEvent recibe un evento del outbox del cliente. Responde 200/201
// tanto si es la primera vez que llega como si ya existía (idempotente) —
// desde la perspectiva del cliente, ambos casos significan "ya quedó
// confirmado, puedo marcarlo como sincronizado y sacarlo de la cola".
func (h *SyncHandler) PushEvent(c *gin.Context) {
	var req dto.SyncEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evento inválido: " + err.Error()})
		return
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}
	entityID, err := uuid.Parse(req.EntityID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_id inválido"})
		return
	}
	clientCreatedAt, err := time.Parse(time.RFC3339, req.ClientCreatedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_created_at inválido, formato esperado RFC3339"})
		return
	}

	event := &domain.SyncEvent{
		ID:              id,
		EntityType:      req.EntityType,
		EntityID:        entityID,
		Operation:       req.Operation,
		Payload:         string(req.Payload),
		Status:          "recibido",
		ClientCreatedAt: clientCreatedAt,
	}

	created, err := h.syncRepo.Create(event)
	if err != nil {
		log.Printf("error guardando evento de sync: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo procesar el evento"})
		return
	}

	if created {
		c.JSON(http.StatusCreated, gin.H{"status": "confirmado", "duplicate": false})
	} else {
		c.JSON(http.StatusOK, gin.H{"status": "confirmado", "duplicate": true})
	}
}