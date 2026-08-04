package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type ReturnHandler struct {
	returnService service.ReturnService
}

func NewReturnHandler(returnService service.ReturnService) *ReturnHandler {
	return &ReturnHandler{returnService: returnService}
}

// RegisterReturn cubre RF-089.
func (h *ReturnHandler) RegisterReturn(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id")) // <- cambia "saleId" por "id"
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	var req dto.RegisterReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	saleItemID, err := uuid.Parse(req.SaleItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sale_item_id inválido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	ret, err := h.returnService.RegisterReturn(saleID, saleItemID, userID, req.Quantity, req.Reason)
	if err != nil {
		log.Printf("error registrando devolución: %v", err)
		switch err {
		case service.ErrNoCashOpen:
			// RNF-009: no se permite devolver sin caja abierta.
			c.JSON(http.StatusForbidden, gin.H{"error": "No se pueden realizar operaciones de devolución sin una caja abierta."})
		case service.ErrSaleNotFound, service.ErrSaleItemNotBelongToSale:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrReturnExceedsQuantity:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la devolución"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": ret.ID.String(), "message": "devolución registrada correctamente"})
}

// ListReturns cubre RF-093.
func (h *ReturnHandler) ListReturns(c *gin.Context) {
	details, err := h.returnService.ListReturns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las devoluciones"})
		return
	}

	response := make([]dto.ReturnResponse, 0, len(details))
	for _, d := range details {
		response = append(response, dto.ReturnResponse{
			ID: d.Return.ID.String(), SaleID: d.Return.SaleID.String(), SaleNumber: d.SaleNumber,
			MedicineName: d.MedicineName, Quantity: d.Return.Quantity, Reason: d.Return.Reason,
			Username: d.Username, CreatedAt: d.Return.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}
