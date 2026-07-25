package handler

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type MedicineHandler struct {
	medicineService service.MedicineService
	batchRepo       repository.BatchRepository
	alertService    *service.AlertService
}

func NewMedicineHandler(medicineService service.MedicineService, batchRepo repository.BatchRepository, alertService *service.AlertService) *MedicineHandler {
	return &MedicineHandler{medicineService: medicineService, batchRepo: batchRepo, alertService: alertService}
}

func (h *MedicineHandler) RegisterMedicine(c *gin.Context) {
	var req dto.RegisterMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	medicine, err := h.medicineService.RegisterMedicine(service.RegisterMedicineInput{
		Barcode: req.Barcode, Name: req.Name, Laboratory: req.Laboratory,
		Concentration: req.Concentration, Presentation: req.Presentation,
		PricePerPill: req.PricePerPill, PricePerSeal: req.PricePerSeal, PricePerBox: req.PricePerBox,
		MinStock: req.MinStock, HasIva: req.HasIva,
		SellsByPill: req.SellsByPill, SellsBySeal: req.SellsBySeal, SellsByBox: req.SellsByBox,
		UnitsPerSeal: req.UnitsPerSeal, UnitsPerBox: req.UnitsPerBox,
	})

	if err != nil {
		log.Printf("error registrando medicamento: %v", err)
		switch err {
		case service.ErrBarcodeTaken:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo registrar el medicamento"})
		}
		return
	}

	c.JSON(http.StatusCreated, toMedicineResponse(medicine))
}

// FindByBarcode: endpoint listo para RF-027, pendiente de probar con el
// lector fisico cuando llegue.
func (h *MedicineHandler) FindByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")

	medicine, err := h.medicineService.FindByBarcode(barcode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no se encontro ningun medicamento con ese codigo de barras"})
		return
	}

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

func (h *MedicineHandler) ListMedicines(c *gin.Context) {
	medicines, err := h.medicineService.ListMedicines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los medicamentos"})
		return
	}

	response := make([]dto.MedicineResponse, 0, len(medicines))
	for i := range medicines {
		response = append(response, toMedicineResponse(&medicines[i]))
	}
	c.JSON(http.StatusOK, response)
}

// UpdateMinStock cubre RF-020 como endpoint dedicado.
func (h *MedicineHandler) UpdateMinStock(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de medicamento invalido"})
		return
	}

	var req dto.UpdateMinStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el stock minimo es obligatorio y debe ser mayor o igual a cero"})
		return
	}

	if err := h.medicineService.UpdateMinStock(id, req.MinStock); err != nil {
		log.Printf("error actualizando stock minimo: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar el stock minimo"})
		return
	}

	// RNF-006: verificar alertas despues de modificar stock minimo
	go h.alertService.CheckInventoryAlerts()

	c.JSON(http.StatusOK, gin.H{"message": "stock minimo actualizado correctamente"})
}

func (h *MedicineHandler) ListCritical(c *gin.Context) {
	critical, err := h.medicineService.ListCriticalMedicines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los medicamentos criticos"})
		return
	}

	response := make([]gin.H, 0, len(critical))
	for _, cr := range critical {
		response = append(response, gin.H{
			"medicine":      toMedicineResponse(&cr.Medicine),
			"current_stock": cr.CurrentStock,
			"min_stock":     cr.Medicine.MinStock,
			"is_critical":   true,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h *MedicineHandler) UpdateAlertDays(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id invalido"})
		return
	}

	var req dto.UpdateAlertDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "days es obligatorio y debe ser mayor a cero"})
		return
	}

	if err := h.medicineService.UpdateAlertDays(id, req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la configuracion de alerta"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "dias de alerta actualizados correctamente"})
}

// UpdateMedicine cubre RF-012: modificar la informacion de un medicamento.
func (h *MedicineHandler) UpdateMedicine(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de medicamento invalido"})
		return
	}

	var req dto.UpdateMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos invalidos: " + err.Error()})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	medicine, err := h.medicineService.UpdateMedicine(id, service.UpdateMedicineInput{
		Name: req.Name, Presentation: req.Presentation, Concentration: req.Concentration,
		PricePerPill: req.PricePerPill, PricePerSeal: req.PricePerSeal, PricePerBox: req.PricePerBox,
		MinStock: req.MinStock, HasIva: req.HasIva,
		SellsByPill: req.SellsByPill, SellsBySeal: req.SellsBySeal, SellsByBox: req.SellsByBox,
		UnitsPerSeal: req.UnitsPerSeal, UnitsPerBox: req.UnitsPerBox,
	}, userID, c.ClientIP())

	if err != nil {
		log.Printf("error actualizando medicamento: %v", err)
		if err == service.ErrMedicineNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar el medicamento"})
		return
	}

	// RNF-006: verificar alertas despues de modificar medicamento
	go h.alertService.CheckInventoryAlerts()

	c.JSON(http.StatusOK, toMedicineResponse(medicine))
}

// DeactivateMedicine cubre RF-013: desactivar un medicamento (soft delete).
func (h *MedicineHandler) DeactivateMedicine(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de medicamento invalido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	if err := h.medicineService.DeactivateMedicine(id, userID, c.ClientIP()); err != nil {
		log.Printf("error desactivando medicamento: %v", err)
		switch err {
		case service.ErrMedicineNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrMedicineAlreadyInactive:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo desactivar el medicamento"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "medicamento desactivado correctamente"})
}

// ActivateMedicine reactiva un medicamento previamente desactivado.
func (h *MedicineHandler) ActivateMedicine(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de medicamento invalido"})
		return
	}

	userIDStr, _ := c.Get("user_id")
	userID, _ := uuid.Parse(userIDStr.(string))

	if err := h.medicineService.ActivateMedicine(id, userID, c.ClientIP()); err != nil {
		log.Printf("error activando medicamento: %v", err)
		switch err {
		case service.ErrMedicineNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrMedicineAlreadyActive:
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo activar el medicamento"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "medicamento activado correctamente"})
}

// SearchMedicines cubre RF-014: busqueda avanzada de medicamentos con
// filtros por nombre, codigo de barras, laboratorio, precios y stock.
func (h *MedicineHandler) SearchMedicines(c *gin.Context) {
	params := service.MedicineSearchParams{
		Name:       c.Query("nombre"),
		Barcode:    c.Query("codigo_barras"),
		Laboratory: c.Query("laboratorio"),
	}

	if minP := c.Query("precio_min"); minP != "" {
		if v, err := strconv.ParseFloat(minP, 64); err == nil {
			params.MinPrice = &v
		}
	}
	if maxP := c.Query("precio_max"); maxP != "" {
		if v, err := strconv.ParseFloat(maxP, 64); err == nil {
			params.MaxPrice = &v
		}
	}

	medicines, err := h.medicineService.SearchMedicines(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo realizar la busqueda"})
		return
	}

	// Enriquecer con existencias totales (RF-014 criterio de aceptacion)
	response := make([]dto.MedicineSearchResponse, 0, len(medicines))
	for _, m := range medicines {
		totalStock, _ := h.batchRepo.SumStockByMedicine(m.ID)

		// Filtro de stock si se especifica
		if hasStockParam := c.Query("con_stock"); hasStockParam == "true" && totalStock <= 0 {
			continue
		}

		response = append(response, dto.MedicineSearchResponse{
			ID:            m.ID.String(),
			Barcode:       m.Barcode,
			Name:          m.Name,
			Laboratory:    m.Laboratory,
			Concentration: m.Concentration,
			Presentation:  m.Presentation,
			PricePerPill:  m.PricePerPill,
			PricePerSeal:  m.PricePerSeal,
			PricePerBox:   m.PricePerBox,
			MinStock:      m.MinStock,
			TotalStock:    totalStock,
			IsActive:      m.IsActive,
			CreatedAt:     m.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, response)
}

func toMedicineResponse(m *domain.Medicine) dto.MedicineResponse {
	return dto.MedicineResponse{
		ID: m.ID.String(), Barcode: m.Barcode, Name: m.Name,
		Laboratory: m.Laboratory, Concentration: m.Concentration, Presentation: m.Presentation,
		PricePerPill: m.PricePerPill, PricePerSeal: m.PricePerSeal, PricePerBox: m.PricePerBox,
		MinStock: m.MinStock, HasIva: m.HasIva,
		SellsByPill: m.SellsByPill, SellsBySeal: m.SellsBySeal, SellsByBox: m.SellsByBox,
		UnitsPerSeal: m.UnitsPerSeal, UnitsPerBox: m.UnitsPerBox,
		IsActive: m.IsActive, CreatedAt: m.CreatedAt.Format(time.RFC3339),
	}
}
