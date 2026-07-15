package handler

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type InventoryHandler struct {
	inventoryService service.InventoryService
}

func NewInventoryHandler(inventoryService service.InventoryService) *InventoryHandler {
	return &InventoryHandler{inventoryService: inventoryService}
}

func (h *InventoryHandler) RegisterIngreso(c *gin.Context) {
	var req dto.RegisterIngresoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	batchID, err := uuid.Parse(req.BatchID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id inválido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	err = h.inventoryService.RegisterIngreso(service.RegisterIngresoInput{
		BatchID: batchID, Quantity: req.Quantity, UserID: userID,
	})
	if err != nil {
		log.Printf("error registrando ingreso: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "ingreso registrado correctamente"})
}
