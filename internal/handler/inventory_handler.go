package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type InventoryHandler struct {
	inventoryService service.InventoryService
	alertService     *service.AlertService
}

func NewInventoryHandler(inventoryService service.InventoryService, alertService *service.AlertService) *InventoryHandler {
	return &InventoryHandler{inventoryService: inventoryService, alertService: alertService}
}

func (h *InventoryHandler) RegisterIngreso(c *gin.Context) {
	var req dto.RegisterIngresoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	batchID, err := uuid.Parse(req.BatchID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id invalido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	err = h.inventoryService.RegisterIngreso(service.RegisterIngresoInput{
		BatchID: batchID, Quantity: req.Quantity, SaleUnit: req.SaleUnit, UserID: userID,
	})
	if err != nil {
		log.Printf("error registrando ingreso: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// RNF-006: verificar alertas despues de registrar ingreso
	go h.alertService.CheckInventoryAlerts()

	c.JSON(http.StatusCreated, gin.H{"message": "ingreso registrado correctamente"})
}

// RegisterAdjustment cubre RF-017: registrar un ajuste manual de inventario.
func (h *InventoryHandler) RegisterAdjustment(c *gin.Context) {
	var req dto.RegisterAdjustmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	medicineID, err := uuid.Parse(req.MedicineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medicine_id invalido"})
		return
	}

	batchID, err := uuid.Parse(req.BatchID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "batch_id invalido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	err = h.inventoryService.RegisterAdjustment(service.RegisterAdjustmentInput{
		MedicineID:  medicineID,
		BatchID:     batchID,
		NewQuantity: req.NewQuantity,
		Reason:      req.Reason,
		UserID:      userID,
	})
	if err != nil {
		log.Printf("error registrando ajuste de inventario: %v", err)
		switch err {
		case service.ErrBatchNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrBatchBelongsToOtherMedicine:
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el ajuste"})
		}
		return
	}

	// RNF-006: verificar alertas despues de registrar ajuste
	go h.alertService.CheckInventoryAlerts()

	c.JSON(http.StatusCreated, gin.H{"message": "ajuste de inventario registrado correctamente"})
}

// ListMovements cubre RF-019: consultar historial de movimientos del inventario.
func (h *InventoryHandler) ListMovements(c *gin.Context) {
	filters := service.MovementFilters{
		MovementType: c.Query("tipo"),
	}

	if fromStr := c.Query("desde"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filters.From = &t
		}
	}
	if toStr := c.Query("hasta"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filters.To = &endOfDay
		}
	}

	movements, err := h.inventoryService.ListMovements(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los movimientos"})
		return
	}

	response := make([]dto.InventoryMovementResponse, 0, len(movements))
	for _, m := range movements {
		batchID := ""
		if m.BatchID != nil {
			batchID = m.BatchID.String()
		}

		response = append(response, dto.InventoryMovementResponse{
			ID:           m.ID.String(),
			MedicineID:   m.MedicineID.String(),
			MedicineName: m.MedicineName,
			BatchID:      batchID,
			BatchCode:    m.BatchCode,
			MovementType: m.MovementType,
			Quantity:     m.Quantity,
			Reason:       m.Reason,
			UserID:       m.UserID.String(),
			Username:     m.Username,
			CreatedAt:    m.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetStockHistory cubre RF-030: consultar historial de existencias con
// filtros por fecha y tipo de movimiento.
func (h *InventoryHandler) GetStockHistory(c *gin.Context) {
	var from, to *time.Time
	movementType := c.Query("tipo")

	if fromStr := c.Query("desde"); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			from = &t
		}
	}
	if toStr := c.Query("hasta"); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			to = &endOfDay
		}
	}

	movements, err := h.inventoryService.GetStockHistory(from, to, movementType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el historial de existencias"})
		return
	}

	response := make([]dto.InventoryMovementResponse, 0, len(movements))
	for _, m := range movements {
		batchID := ""
		if m.BatchID != nil {
			batchID = m.BatchID.String()
		}

		response = append(response, dto.InventoryMovementResponse{
			ID:           m.ID.String(),
			MedicineID:   m.MedicineID.String(),
			MedicineName: m.MedicineName,
			BatchID:      batchID,
			BatchCode:    m.BatchCode,
			MovementType: m.MovementType,
			Quantity:     m.Quantity,
			Reason:       m.Reason,
			UserID:       m.UserID.String(),
			Username:     m.Username,
			CreatedAt:    m.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetGeneralInventory cubre RF-028: inventario general consolidado
// por medicamento.
func (h *InventoryHandler) GetGeneralInventory(c *gin.Context) {
	items, err := h.inventoryService.GetGeneralInventory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo cargar el inventario general"})
		return
	}

	// Aplicar filtros opcionales por query params
	nameFilter := c.Query("nombre")
	orderBy := c.Query("ordenar_por")

	response := make([]dto.InventoryGeneralItem, 0, len(items))
	for _, item := range items {
		// Filtro por nombre si se especifica
		if nameFilter != "" {
			if !containsInsensitive(item.Medicine.Name, nameFilter) {
				continue
			}
		}

		status := "activo"
		if !item.Medicine.IsActive {
			status = "inactivo"
		}

		nextExp := ""
		if item.NextExpiration != nil {
			nextExp = item.NextExpiration.Format("2006-01-02")
		}

		response = append(response, dto.InventoryGeneralItem{
			ID:               item.Medicine.ID.String(),
			Barcode:          item.Medicine.Barcode,
			Name:             item.Medicine.Name,
			Laboratory:       item.Medicine.Laboratory,
			TotalStock:       item.TotalStock,
			MinStock:         item.Medicine.MinStock,
			IsActive:         item.Medicine.IsActive,
			Status:           status,
			NextExpiration:   nextExp,
			AvailableBatches: item.AvailableBatches,
		})
	}

	// Ordenamiento por query param
	if orderBy == "stock_asc" {
		sortByStockAsc(response)
	} else if orderBy == "stock_desc" {
		sortByStockDesc(response)
	} else if orderBy == "nombre" {
		sortByName(response)
	}

	c.JSON(http.StatusOK, response)
}

// GetCriticalInventory cubre RF-029: medicamentos criticos (stock bajo,
// proximo a vencer, vencidos).
func (h *InventoryHandler) GetCriticalInventory(c *gin.Context) {
	days := 30
	if d := c.Query("dias_vencimiento"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	items, err := h.inventoryService.GetCriticalInventory(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los medicamentos criticos"})
		return
	}

	response := make([]dto.CriticalInventoryItem, 0, len(items))
	for _, item := range items {
		response = append(response, dto.CriticalInventoryItem{
			ID:         item.Medicine.ID.String(),
			Barcode:    item.Medicine.Barcode,
			Name:       item.Medicine.Name,
			Laboratory: item.Medicine.Laboratory,
			TotalStock: item.TotalStock,
			MinStock:   item.Medicine.MinStock,
			IsActive:   item.Medicine.IsActive,
			Reasons:    item.Reasons,
		})
	}

	c.JSON(http.StatusOK, response)
}

// --- Funciones auxiliares de ordenamiento ---

func containsInsensitive(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(substr) == 0 ||
			findInsensitive(s, substr))
}

func findInsensitive(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if equalFoldSlice(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func equalFoldSlice(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func sortByStockAsc(items []dto.InventoryGeneralItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].TotalStock < items[j-1].TotalStock; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func sortByStockDesc(items []dto.InventoryGeneralItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].TotalStock > items[j-1].TotalStock; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func sortByName(items []dto.InventoryGeneralItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].Name < items[j-1].Name; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
