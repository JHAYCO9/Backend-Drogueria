package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type CashRegisterHandler struct {
	cashService service.CashRegisterService
	userRepo    repository.UserRepository
}

func NewCashRegisterHandler(cashService service.CashRegisterService, userRepo repository.UserRepository) *CashRegisterHandler {
	return &CashRegisterHandler{cashService: cashService, userRepo: userRepo}
}

func (h *CashRegisterHandler) OpenCashRegister(c *gin.Context) {
	var req dto.OpenCashRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el valor inicial es obligatorio y debe ser mayor o igual a cero"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	cr, err := h.cashService.OpenCashRegister(userID, req.OpeningAmount)
	if err != nil {
		log.Printf("error abriendo caja: %v", err)
		if err == service.ErrCashAlreadyOpen {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la caja"})
		return
	}

	c.JSON(http.StatusCreated, dto.OpenCashRegisterResponse{
		ID: cr.ID.String(), OpeningAmount: cr.OpeningAmount, OpenedAt: cr.OpenedAt.Format(time.RFC3339),
	})
}

func (h *CashRegisterHandler) GetStatus(c *gin.Context) {
	status, err := h.cashService.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo consultar el estado de la caja"})
		return
	}

	if !status.IsOpen {
		c.JSON(http.StatusOK, dto.CashRegisterStatusResponse{IsOpen: false})
		return
	}

	c.JSON(http.StatusOK, dto.CashRegisterStatusResponse{
		IsOpen: true, ID: status.CashRegister.ID.String(),
		OpeningAmount: status.CashRegister.OpeningAmount,
		ExpectedCash:  status.ExpectedCash, TotalNequiSales: status.TotalNequiSales,
		CurrentBalance: status.CurrentBalance,
		OpenedAt:       status.CashRegister.OpenedAt.Format(time.RFC3339), OpenedByName: status.OpenedByName,
	})
}

// CloseCashRegister cubre RF-052 + RF-053 + RF-054.
func (h *CashRegisterHandler) CloseCashRegister(c *gin.Context) {
	var req dto.CloseCashRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el valor contado es obligatorio y debe ser mayor o igual a cero"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	cr, err := h.cashService.CloseCashRegister(userID, req.CountedAmount, req.DifferenceReason)
	if err != nil {
		log.Printf("error cerrando caja: %v", err)
		if err == service.ErrCashNotOpen {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cerrar la caja"})
		return
	}

	c.JSON(http.StatusOK, dto.CloseCashRegisterResponse{
		ExpectedAmount: *cr.ExpectedAmount, CountedAmount: *cr.CountedAmount,
		DifferenceAmount: *cr.DifferenceAmount, ClosedAt: cr.ClosedAt.Format(time.RFC3339),
	})
}

// GetMovements cubre RF-051. Query params opcionales: ?user_id=...&from=AAAA-MM-DD&to=AAAA-MM-DD
func (h *CashRegisterHandler) GetMovements(c *gin.Context) {
	var filter service.CashMovementFilterInput

	if uid := c.Query("user_id"); uid != "" {
		if parsed, err := uuid.Parse(uid); err == nil {
			filter.UserID = &parsed
		}
	}
	if fromStr := c.Query("from"); fromStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromStr); err == nil {
			filter.From = &parsed
		}
	}
	if toStr := c.Query("to"); toStr != "" {
		if parsed, err := time.Parse("2006-01-02", toStr); err == nil {
			end := parsed.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.To = &end
		}
	}

	movements, err := h.cashService.ListMovements(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los movimientos"})
		return
	}

	response := make([]dto.CashMovementResponse, 0, len(movements))
	for _, m := range movements {
		username := ""
		if u, err := h.userRepo.FindByID(m.UserID); err == nil {
			username = u.FullName
		}
		response = append(response, dto.CashMovementResponse{
			ID: m.ID.String(), MovementType: m.MovementType, Amount: m.Amount,
			Description: m.Description, Username: username,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}
