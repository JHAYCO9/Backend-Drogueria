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

// ─────────────────────────────────────────────────────────────
// RF-016 + RNF-008 + RNF-012: POST /api/v1/cash/open
// ─────────────────────────────────────────────────────────────

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
			// RNF-008: HTTP 400 Bad Request cuando ya existe una caja abierta.
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo abrir la caja"})
		return
	}

	c.JSON(http.StatusCreated, dto.OpenCashRegisterResponse{
		ID: cr.ID.String(), OpeningAmount: cr.OpeningAmount, OpenedAt: cr.OpenedAt.Format(time.RFC3339),
	})
}

// ─────────────────────────────────────────────────────────────
// RF-049: GET /api/v1/cash-register/status (existente — sin cambios)
// ─────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────
// RF-019: GET /api/v1/cash/status — arqueo detallado
// ─────────────────────────────────────────────────────────────

// GetDetailedStatus devuelve el arqueo completo en tiempo real (RF-019):
// totales de ventas por método de pago, movimientos manuales y saldo teórico.
func (h *CashRegisterHandler) GetDetailedStatus(c *gin.Context) {
	detail, err := h.cashService.GetDetailedStatus()
	if err != nil {
		if err == service.ErrCashNotOpen {
			c.JSON(http.StatusOK, dto.CashStatusDetailResponse{IsOpen: false})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo consultar el estado de la caja"})
		return
	}

	movItems := make([]dto.ManualMovementItem, 0, len(detail.MovimientosManuales))
	for _, m := range detail.MovimientosManuales {
		movType := "ingreso_manual"
		movAmount := m.Amount
		if m.Amount < 0 || m.MovementType == "egreso" || m.MovementType == "egreso_manual" {
			movType = "egreso_manual"
			if movAmount < 0 {
				movAmount = -movAmount
			}
		}
		movItems = append(movItems, dto.ManualMovementItem{
			ID:        m.ID.String(),
			Type:      movType,
			Amount:    movAmount,
			Reason:    m.Reason,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, dto.CashStatusDetailResponse{
		IsOpen:                 true,
		SessionID:              detail.CashRegister.ID.String(),
		MontoInicial:           detail.CashRegister.OpeningAmount,
		FechaApertura:          detail.CashRegister.OpenedAt.Format(time.RFC3339),
		AbiertoPor:             detail.OpenedByName,
		TotalVentasEfectivo:    detail.TotalVentasEfectivo,
		TotalVentasOtrosMedios: detail.TotalVentasOtrosMedios,
		TotalIngresosManual:    detail.TotalIngresosManual,
		TotalEgresosManual:     detail.TotalEgresosManual,
		SaldoTeorico:           detail.SaldoTeorico,
		ListaMovimientos:       movItems,
	})
}

// ─────────────────────────────────────────────────────────────
// RF-017 + RNF-011 + RNF-012: POST /api/v1/cash/close
// ─────────────────────────────────────────────────────────────

// CloseCashRegister cubre RF-017 + RF-052 + RF-053 + RF-054.
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

	discrepancyType := "NONE"
	if cr.DiscrepancyType != nil {
		discrepancyType = *cr.DiscrepancyType
	}

	c.JSON(http.StatusOK, dto.CloseCashRegisterResponse{
		ExpectedAmount:   *cr.ExpectedAmount,
		CountedAmount:    *cr.CountedAmount,
		DifferenceAmount: *cr.DifferenceAmount,
		DiscrepancyType:  discrepancyType,
		ClosedAt:         cr.ClosedAt.Format(time.RFC3339),
	})
}

// ─────────────────────────────────────────────────────────────
// RF-018 + RNF-010: POST /api/v1/cash/movements
// ─────────────────────────────────────────────────────────────

// RegisterManualMovement registra un ingreso o egreso manual de caja (RF-018).
// RNF-010: este endpoint solo permite CREATE. No existe PUT ni DELETE.
func (h *CashRegisterHandler) RegisterManualMovement(c *gin.Context) {
	var req dto.RegisterManualMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "datos inválidos: tipo debe ser 'ingreso_manual' o 'egreso_manual', monto > 0 y justificación obligatoria (mín. 3 caracteres)",
		})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	mov, err := h.cashService.RegisterManualMovement(userID, req.Type, req.Reason, req.Amount)
	if err != nil {
		log.Printf("error registrando movimiento manual: %v", err)
		switch err {
		case service.ErrCashNotOpen:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrInvalidMovementAmount, service.ErrMovementReasonRequired:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el movimiento"})
		}
		return
	}

	c.JSON(http.StatusCreated, dto.RegisterManualMovementResponse{
		ID:        mov.ID.String(),
		Type:      req.Type,
		Amount:    mov.Amount,
		Reason:    mov.Reason,
		CreatedAt: mov.CreatedAt.Format(time.RFC3339),
	})
}

// ─────────────────────────────────────────────────────────────
// RF-051: GET /api/v1/cash-register/movements (existente)
// ─────────────────────────────────────────────────────────────

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
		movType := m.MovementType
		movAmount := m.Amount
		if m.MovementCategory == "manual" {
			if m.Amount < 0 || m.MovementType == "egreso" || m.MovementType == "egreso_manual" {
				movType = "egreso_manual"
				if movAmount < 0 {
					movAmount = -movAmount
				}
			} else {
				movType = "ingreso_manual"
			}
		}
		response = append(response, dto.CashMovementResponse{
			ID: m.ID.String(), MovementType: movType, MovementCategory: m.MovementCategory,
			Amount: movAmount, Description: m.Description, Reason: m.Reason,
			Username: username, CreatedAt: m.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}

// ─────────────────────────────────────────────────────────────
// RF-056: GET /api/v1/cash-register/history
// Consultar historial de cierres de caja (filtros: from, to, user_id)
// ─────────────────────────────────────────────────────────────

func (h *CashRegisterHandler) ListClosedRegisters(c *gin.Context) {
	var from, to *time.Time
	var userID *uuid.UUID

	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "formato de fecha 'from' inválido, use YYYY-MM-DD"})
			return
		}
		from = &t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "formato de fecha 'to' inválido, use YYYY-MM-DD"})
			return
		}
		end := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		to = &end
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id inválido"})
			return
		}
		userID = &uid
	}

	details, err := h.cashService.ListClosedRegisters(from, to, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el historial de cierres"})
		return
	}

	response := make([]dto.ClosedRegisterResponse, 0, len(details))
	for _, d := range details {
		closedAtStr := ""
		if d.Register.ClosedAt != nil {
			closedAtStr = d.Register.ClosedAt.Format(time.RFC3339)
		}
		expectedAmount := 0.0
		if d.Register.ExpectedAmount != nil {
			expectedAmount = *d.Register.ExpectedAmount
		}
		countedAmount := 0.0
		if d.Register.CountedAmount != nil {
			countedAmount = *d.Register.CountedAmount
		}
		differenceAmount := 0.0
		if d.Register.DifferenceAmount != nil {
			differenceAmount = *d.Register.DifferenceAmount
		}
		discrepancyType := ""
		if d.Register.DiscrepancyType != nil {
			discrepancyType = *d.Register.DiscrepancyType
		}
		differenceReason := ""
		if d.Register.DifferenceReason != nil {
			differenceReason = *d.Register.DifferenceReason
		}

		response = append(response, dto.ClosedRegisterResponse{
			ID:               d.Register.ID.String(),
			OpenedAt:         d.Register.OpenedAt.Format(time.RFC3339),
			ClosedAt:         closedAtStr,
			OpenedByName:     d.OpenedByName,
			ClosedByName:     d.ClosedByName,
			OpeningAmount:    d.Register.OpeningAmount,
			CountedAmount:    countedAmount,
			ExpectedAmount:   expectedAmount,
			DifferenceAmount: differenceAmount,
			DiscrepancyType:  discrepancyType,
			DifferenceReason: differenceReason,
			TotalVentasEfectivo: d.TotalVentasCash,
			TotalOtrosMedios:    d.TotalOtros,
			TotalIngresos:       d.TotalIngresos,
			TotalEgresos:        d.TotalEgresos,
			SaldoTeorico:        d.SaldoTeorico,
		})
	}

	c.JSON(http.StatusOK, response)
}
