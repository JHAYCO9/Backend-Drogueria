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
	userRepo repository.UserRepository
}

func NewSaleHandler(saleService service.SaleService, medicineRepo interface {
	FindByID(id uuid.UUID) (*domain.Medicine, error)
}, userRepo repository.UserRepository) *SaleHandler {
	return &SaleHandler{saleService: saleService, medicineRepo: medicineRepo, userRepo: userRepo}
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
		items = append(items, dto.SaleListItem{
			ID: sale.ID.String(), SaleNumber: sale.SaleNumber, Status: sale.Status,
			TotalAmount: sale.TotalAmount, PaymentMethod: sale.PaymentMethod,
			CashierName: cashierName, CreatedAt: sale.CreatedAt.Format(time.RFC3339),
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
		itemResponses = append(itemResponses, dto.SaleItemResponse{
			ID: it.ID.String(), MedicineID: it.MedicineID.String(), MedicineName: medicineName,
			BatchID: it.BatchID.String(), Quantity: it.Quantity, SaleUnit: it.SaleUnit,
			UnitPrice: it.UnitPrice, Subtotal: it.Subtotal, TaxAmount: it.TaxAmount,
		})
		totalTax += it.TaxAmount
	}

	var finalizedAt *string
	if sale.FinalizedAt != nil {
		f := sale.FinalizedAt.Format(time.RFC3339)
		finalizedAt = &f
	}

	return &dto.SaleResponse{
		ID: sale.ID.String(), SaleNumber: sale.SaleNumber, Status: sale.Status,
		Items: itemResponses, TotalAmount: sale.TotalAmount, TotalTaxAmount: totalTax,
		PaymentMethod: sale.PaymentMethod, AmountReceived: sale.AmountReceived, ChangeGiven: sale.ChangeGiven,
		CreatedAt: sale.CreatedAt.Format(time.RFC3339), FinalizedAt: finalizedAt,
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
