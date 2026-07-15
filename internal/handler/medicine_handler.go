package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type MedicineHandler struct {
	medicineService service.MedicineService
}

func NewMedicineHandler(medicineService service.MedicineService) *MedicineHandler {
	return &MedicineHandler{medicineService: medicineService}
}

func (h *MedicineHandler) RegisterMedicine(c *gin.Context) {
	var req dto.RegisterMedicineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	medicine, err := h.medicineService.RegisterMedicine(service.RegisterMedicineInput{
		Barcode:       req.Barcode,
		Name:          req.Name,
		Laboratory:    req.Laboratory,
		Concentration: req.Concentration,
		Presentation:  req.Presentation,
		PricePerPill:  req.PricePerPill,
		PricePerSeal:  req.PricePerSeal,
		PricePerBox:   req.PricePerBox,
		MinStock:      req.MinStock,
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
// lector físico cuando llegue. Mientras tanto se puede probar en Postman
// mandando el código a mano — el comportamiento es idéntico, ya que un
// lector USB tipo "teclado" simplemente escribe el código y un Enter,
// nada distinto a teclearlo manualmente desde la perspectiva del backend.
func (h *MedicineHandler) FindByBarcode(c *gin.Context) {
	barcode := c.Param("barcode")

	medicine, err := h.medicineService.FindByBarcode(barcode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no se encontró ningún medicamento con ese código de barras"})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "id de medicamento inválido"})
		return
	}

	var req dto.UpdateMinStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "el stock mínimo es obligatorio y debe ser mayor o igual a cero"})
		return
	}

	if err := h.medicineService.UpdateMinStock(id, req.MinStock); err != nil {
		log.Printf("error actualizando stock mínimo: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar el stock mínimo"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "stock mínimo actualizado correctamente"})
}

func (h *MedicineHandler) ListCritical(c *gin.Context) {
	critical, err := h.medicineService.ListCriticalMedicines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los medicamentos críticos"})
		return
	}

	response := make([]gin.H, 0, len(critical))
	for _, c := range critical {
		response = append(response, gin.H{
			"medicine":      toMedicineResponse(&c.Medicine),
			"current_stock": c.CurrentStock,
			"min_stock":     c.Medicine.MinStock,
			"is_critical":   true,
		})
	}
	c.JSON(http.StatusOK, response)
}

func (h *MedicineHandler) UpdateAlertDays(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
		return
	}

	var req dto.UpdateAlertDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "days es obligatorio y debe ser mayor a cero"})
		return
	}

	if err := h.medicineService.UpdateAlertDays(id, req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo actualizar la configuración de alerta"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "días de alerta actualizados correctamente"})
}	

func toMedicineResponse(m *domain.Medicine) dto.MedicineResponse {
	return dto.MedicineResponse{
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
		IsActive:      m.IsActive,
		CreatedAt:     m.CreatedAt.Format(time.RFC3339),
	}
}
