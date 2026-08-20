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
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type SaleHandler struct {
	saleService  service.SaleService
	medicineRepo interface {
		FindByID(id uuid.UUID) (*domain.Medicine, error)
	}
	returnRepo repository.ReturnRepository
	userRepo   repository.UserRepository
}

func NewSaleHandler(saleService service.SaleService, medicineRepo interface {
	FindByID(id uuid.UUID) (*domain.Medicine, error)
}, returnRepo repository.ReturnRepository, userRepo repository.UserRepository) *SaleHandler {
	return &SaleHandler{saleService: saleService, medicineRepo: medicineRepo, returnRepo: returnRepo, userRepo: userRepo}
}

// GetReceipt cubre RF-043.
func (h *SaleHandler) GetReceipt(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	receipt, err := h.saleService.GetReceipt(saleID)
	if err != nil {
		log.Printf("error generando comprobante: %v", err)
		if err == service.ErrSaleNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	items := make([]dto.ReceiptItemLine, 0, len(receipt.Items))
	for _, it := range receipt.Items {
		items = append(items, dto.ReceiptItemLine{
			MedicineName: it.MedicineName, Quantity: it.Quantity, SaleUnit: it.SaleUnit,
			UnitPrice: it.UnitPrice, BasePrice: it.BasePrice, TaxAmount: it.TaxAmount,
			HasIva: it.HasIva, Subtotal: it.Subtotal,
		})
	}

	c.JSON(http.StatusOK, dto.ReceiptResponse{
		SaleNumber:     receipt.SaleNumber,
		Date:           receipt.CreatedAt.Format("2006-01-02"),
		Time:           receipt.CreatedAt.Format("15:04"),
		CashierName:    receipt.CashierName,
		Items:          items,
		TotalAmount:    receipt.TotalAmount,
		TotalTax:       receipt.TotalTax,
		PaymentMethod:  receipt.PaymentMethod,
		AmountReceived: receipt.AmountReceived,
		ChangeGiven:    receipt.ChangeGiven,
	})
}

// ListSales cubre RF-044.
func (h *SaleHandler) ListSales(c *gin.Context) {
	sales, err := h.saleService.ListSales()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el historial de ventas"})
		return
	}

	c.JSON(http.StatusOK, h.toSaleListItems(sales))
}

// GetSalesByDateRange cubre RF-047. Espera ?from=2026-01-01&to=2026-01-31.
func (h *SaleHandler) GetSalesByDateRange(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "los parámetros from y to son obligatorios (formato AAAA-MM-DD)"})
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha 'from' inválida, formato esperado AAAA-MM-DD"})
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fecha 'to' inválida, formato esperado AAAA-MM-DD"})
		return
	}
	// incluye todo el día final, no solo las 00:00
	to = to.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	result, err := h.saleService.GetSalesByDateRange(from, to)
	if err != nil {
		log.Printf("error consultando ventas por rango: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo consultar el rango de fechas"})
		return
	}

	c.JSON(http.StatusOK, dto.SalesDateRangeSummary{
		FromDate: fromStr, ToDate: toStr,
		SalesCount: result.SalesCount, TotalSold: result.TotalSold, AverageSale: result.AverageSale,
		Sales: h.toSaleListItems(result.Sales),
	})
}

func (h *SaleHandler) toSaleListItems(sales []domain.Sale) []dto.SaleListItem {
	items := make([]dto.SaleListItem, 0, len(sales))
	for _, sale := range sales {
		cashierName := ""
		if u, err := h.userRepo.FindByID(sale.UserID); err == nil {
			cashierName = u.FullName
		}
		hasReturns, _ := h.returnRepo.ExistsForSale(sale.ID)
		items = append(items, dto.SaleListItem{
			ID: sale.ID.String(), SaleNumber: sale.SaleNumber, Status: sale.Status,
			TotalAmount: sale.TotalAmount, PaymentMethod: sale.PaymentMethod,
			CashierName: cashierName, CreatedAt: service.FormatBogota(sale.CreatedAt),
			HasReturns: hasReturns,
		})
	}
	return items
}

// StartSale cubre RF-031 + RF-032.
func (h *SaleHandler) StartSale(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "usuario no identificado"})
		return
	}

	sale, err := h.saleService.StartSale(userID)
	if err != nil {
		if err == service.ErrNoCashOpen {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		log.Printf("error iniciando venta: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo iniciar la venta"})
		return
	}

	response, err := h.buildSaleResponse(sale.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el detalle de la venta"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// buildSaleResponse arma el dto.SaleResponse completo para una venta
// (usado por StartSale al retomar, y por GetSale).
func (h *SaleHandler) buildSaleResponse(saleID uuid.UUID) (*dto.SaleResponse, error) {
	sale, items, err := h.saleService.GetSaleDetail(saleID)
	if err != nil {
		return nil, err
	}

	itemResponses := make([]dto.SaleItemResponse, 0, len(items))
	totalTax := 0.0
	for _, it := range items {
		medicineName := ""
		if m, err := h.medicineRepo.FindByID(it.MedicineID); err == nil {
			medicineName = m.Name
		}
		returnedQty, _ := h.returnRepo.SumReturnedForSaleItem(it.ID)

		itemResponses = append(itemResponses, dto.SaleItemResponse{
			ID: it.ID.String(), MedicineID: it.MedicineID.String(), MedicineName: medicineName,
			BatchID: it.BatchID.String(), Quantity: it.Quantity, SaleUnit: it.SaleUnit,
			UnitPrice: it.UnitPrice, Subtotal: it.Subtotal, TaxAmount: it.TaxAmount,
			ReturnedQuantity: returnedQty,
		})
		totalTax += it.TaxAmount
	}

	var finalizedAt *string
	if sale.FinalizedAt != nil {
		f := service.FormatBogota(*sale.FinalizedAt)
		finalizedAt = &f
	}

	return &dto.SaleResponse{
		ID: sale.ID.String(), SaleNumber: sale.SaleNumber, Status: sale.Status,
		Items: itemResponses, TotalAmount: sale.TotalAmount, TotalTaxAmount: totalTax,
		PaymentMethod: sale.PaymentMethod, AmountReceived: sale.AmountReceived, ChangeGiven: sale.ChangeGiven,
		CreatedAt: service.FormatBogota(sale.CreatedAt), FinalizedAt: finalizedAt,
	}, nil
}

// AddItem cubre RF-033.
func (h *SaleHandler) AddItem(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	var req dto.AddSaleItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	input := service.AddSaleItemInput{Barcode: req.Barcode, Quantity: req.Quantity, SaleUnit: req.SaleUnit}
	if req.MedicineID != "" {
		mid, err := uuid.Parse(req.MedicineID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "medicine_id inválido"})
			return
		}
		input.MedicineID = mid
	}

	_, err = h.saleService.AddItem(saleID, input)
	if err != nil {
		log.Printf("error agregando ítem a venta: %v", err)
		switch err {
		case service.ErrSaleNotFound, service.ErrMedicineNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress, service.ErrNoStockAvailable:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress, service.ErrNoStockAvailable, service.ErrInvalidSaleUnit:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	h.GetSale(c) // devuelve la venta completa actualizada
}

// RemoveItem cubre RF-035.
func (h *SaleHandler) RemoveItem(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de ítem inválido"})
		return
	}

	if err := h.saleService.RemoveItem(saleID, itemID); err != nil {
		switch err {
		case service.ErrSaleNotFound, service.ErrItemNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo eliminar el producto"})
		}
		return
	}

	h.GetSale(c)
}

// UpdateItem cubre RF-034: PUT /api/v1/sales/:id/items/:itemId
func (h *SaleHandler) UpdateItem(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}
	itemID, err := uuid.Parse(c.Param("itemId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de ítem inválido"})
		return
	}

	var req struct {
		Quantity int `json:"quantity" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "la cantidad debe ser mayor a cero"})
		return
	}

	if _, err := h.saleService.UpdateItemQuantity(saleID, itemID, req.Quantity); err != nil {
		switch err {
		case service.ErrSaleNotFound, service.ErrItemNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			// RF-037: Si stock_disponible < cantidad_solicitada -> HTTP 400 Bad Request
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}

	h.GetSale(c)
}

// CancelSale cubre RF-036.
func (h *SaleHandler) CancelSale(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	if err := h.saleService.CancelSale(saleID); err != nil {
		switch err {
		case service.ErrSaleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cancelar la venta"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "venta cancelada correctamente"})
}

// FinalizeCash cubre RF-039 + RF-041.
func (h *SaleHandler) FinalizeCash(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	var req dto.FinalizeCashRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el valor recibido es obligatorio y debe ser mayor a cero"})
		return
	}

	sale, change, err := h.saleService.FinalizeCash(saleID, req.AmountReceived)
	if err != nil {
		log.Printf("error finalizando venta en efectivo: %v", err)
		switch err {
		case service.ErrSaleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress, service.ErrEmptySale, service.ErrInsufficientPayment:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo finalizar la venta"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "venta finalizada correctamente",
		"sale_id":      sale.ID.String(),
		"sale_number":  sale.SaleNumber,
		"total_amount": sale.TotalAmount,
		"change_given": change,
	})
}

// FinalizeNequi cubre RF-040 + RF-041.
func (h *SaleHandler) FinalizeNequi(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	var req dto.FinalizeNequiRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el valor pagado es obligatorio y debe ser mayor a cero"})
		return
	}

	sale, err := h.saleService.FinalizeNequi(saleID, req.AmountPaid)
	if err != nil {
		log.Printf("error finalizando venta por Nequi: %v", err)
		switch err {
		case service.ErrSaleNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrSaleNotInProgress, service.ErrEmptySale:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo finalizar la venta"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "venta finalizada correctamente",
		"sale_id":      sale.ID.String(),
		"sale_number":  sale.SaleNumber,
		"total_amount": sale.TotalAmount,
	})
}

// GetSale devuelve la venta con su detalle completo — la usan AddItem y
// RemoveItem para responder siempre con el estado actualizado.
func (h *SaleHandler) GetSale(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de venta inválido"})
		return
	}

	response, err := h.buildSaleResponse(saleID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "venta no encontrada"})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ============================================================
// Sprint 3 — Nuevos handlers
// ============================================================

// CreateAtomicSale cubre RF-021: POST /api/v1/sales/atomic
// Registra una venta completa en una sola transacción ACID.
func (h *SaleHandler) CreateAtomicSale(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Usuario no identificado",
			"data":    nil,
			"errors":  "token sin user_id válido",
		})
		return
	}

	var req dto.CreateSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Datos de entrada inválidos",
			"data":    nil,
			"errors":  err.Error(),
		})
		return
	}

	result, err := h.saleService.CreateSaleAtomic(userID, req)
	if err != nil {
		log.Printf("error en venta atómica: %v", err)
		switch err {
		case service.ErrNoCashOpen:
			c.JSON(http.StatusConflict, gin.H{
				"success": false, "message": err.Error(), "data": nil, "errors": err.Error(),
			})
		case service.ErrMedicineNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"success": false, "message": err.Error(), "data": nil, "errors": err.Error(),
			})
		case service.ErrInsufficientPayment, service.ErrInvalidSaleUnit:
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"success": false, "message": err.Error(), "data": nil, "errors": err.Error(),
			})
		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false, "message": "No se pudo procesar la venta", "data": nil, "errors": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Venta registrada exitosamente",
		"data":    result,
		"errors":  nil,
	})
}

// VoidCompletedSale cubre RF-023: POST /api/v1/sales/:id/void
// Solo accesible para Administradores (middleware RequireRole en el router).
func (h *SaleHandler) VoidCompletedSale(c *gin.Context) {
	saleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false, "message": "ID de venta inválido", "data": nil,
			"errors": "el parámetro 'id' debe ser un UUID válido",
		})
		return
	}

	adminIDStr, _ := c.Get("user_id")
	adminID, err := uuid.Parse(adminIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false, "message": "Administrador no identificado", "data": nil, "errors": "token inválido",
		})
		return
	}

	var req dto.CancelSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false, "message": "El motivo de anulación es obligatorio (mínimo 5 caracteres)",
			"data": nil, "errors": err.Error(),
		})
		return
	}

	if err := h.saleService.VoidCompletedSale(saleID, adminID, req.Reason); err != nil {
		log.Printf("error anulando venta %s: %v", saleID, err)
		switch err {
		case service.ErrSaleNotFound:
			c.JSON(http.StatusNotFound, gin.H{
				"success": false, "message": err.Error(), "data": nil, "errors": err.Error(),
			})
		case service.ErrSaleAlreadyCancelled, service.ErrSaleNotFinalized:
			c.JSON(http.StatusConflict, gin.H{
				"success": false, "message": err.Error(), "data": nil, "errors": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false, "message": "No se pudo anular la venta", "data": nil, "errors": err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Venta anulada correctamente. Stock reintegrado.",
		"data":    gin.H{"sale_id": saleID.String()},
		"errors":  nil,
	})
}

// ListSalesFiltered cubre RF-022: GET /api/v1/sales (con filtros + paginación).
// Acepta query params: start_date, end_date, customer_id, user_id, status, page, page_size.
func (h *SaleHandler) ListSalesFiltered(c *gin.Context) {
	params := repository.SaleFilterParams{
		Status:   c.Query("status"),
		Page:     1,
		PageSize: 20,
	}

	if v := c.Query("start_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			params.StartDate = &t
		}
	}
	if v := c.Query("end_date"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			end := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			params.EndDate = &end
		}
	}
	if v := c.Query("customer_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			params.CustomerID = &id
		}
	}
	if v := c.Query("user_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			params.UserID = &id
		}
	}

	sales, total, err := h.saleService.ListSalesFiltered(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false, "message": "No se pudo obtener el listado de ventas",
			"data": nil, "errors": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Ventas obtenidas exitosamente",
		"data": gin.H{
			"total":     total,
			"page":      params.Page,
			"page_size": params.PageSize,
			"sales":     h.toSaleListItems(sales),
		},
		"errors": nil,
	})
}
