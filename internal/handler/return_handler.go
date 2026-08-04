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

// RegisterReturn cubre RF-089 + RF-090 + RF-092.
func (h *ReturnHandler) RegisterReturn(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	var req dto.RegisterReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "datos inválidos: quantity debe ser mayor a 0 y reason es obligatorio",
		})
		return
	}

	// RF-092: validar que el motivo sea uno de los predefinidos
	if !dto.IsValidReturnReason(req.Reason) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":           "motivo de devolución inválido. Debe ser uno de los motivos permitidos",
			"motivos_validos": dto.ValidReturnReasons,
		})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "No se pueden realizar operaciones de devolución sin una caja abierta."})
		case service.ErrSaleNotFound, service.ErrSaleItemNotBelongToSale:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrReturnExceedsQuantity:
			// RF-090: mensaje específico cuando se supera la cantidad vendida
			c.JSON(http.StatusConflict, gin.H{
				"error": "La cantidad a devolver supera las unidades registradas en la venta para este producto. Verifique la cantidad e intente nuevamente.",
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar la devolución"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": ret.ID.String(), "message": "devolución registrada correctamente"})
}

// VoidReturn cubre RF-094: anular una devolución previa por parte del Administrador (RNF-029).
func (h *ReturnHandler) VoidReturn(c *gin.Context) {
	returnID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de devolución inválido"})
		return
	}

	var req dto.VoidReturnRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "motivo de anulación obligatorio (mín. 3 caracteres)"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	ret, err := h.returnService.VoidReturn(returnID, userID, req.Reason)
	if err != nil {
		log.Printf("error anulando devolución: %v", err)
		switch err {
		case service.ErrReturnNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrReturnAlreadyVoided:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        ret.ID.String(),
		"message":   "devolución anulada correctamente y stock de inventario revertido",
		"is_voided": ret.IsVoided,
	})
}

// ListReturns cubre RF-093 y RNF-027 (permanencia del historial).
func (h *ReturnHandler) ListReturns(c *gin.Context) {
	details, err := h.returnService.ListReturns()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar las devoluciones"})
		return
	}

	response := make([]dto.ReturnResponse, 0, len(details))
	for _, d := range details {
		voidedAtStr := ""
		if d.Return.VoidedAt != nil {
			voidedAtStr = d.Return.VoidedAt.Format(time.RFC3339)
		}
		response = append(response, dto.ReturnResponse{
			ID: d.Return.ID.String(), SaleID: d.Return.SaleID.String(), SaleNumber: d.SaleNumber,
			MedicineName: d.MedicineName, Quantity: d.Return.Quantity, Reason: d.Return.Reason,
			Username: d.Username, IsVoided: d.Return.IsVoided, VoidedReason: d.Return.VoidedReason,
			VoidedAt: voidedAtStr, VoidedByName: d.VoidedByName,
			CreatedAt: d.Return.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}
