package handler

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
	"github.com/drogueria/drogueria-micky-backend/internal/service"
)

type BatchHandler struct {
	batchService    service.BatchService
	medicineService service.MedicineService
	medicineRepo    repository.MedicineRepository
}

func NewBatchHandler(batchService service.BatchService, medicineService service.MedicineService, medicineRepo repository.MedicineRepository) *BatchHandler {
	return &BatchHandler{batchService: batchService, medicineService: medicineService, medicineRepo: medicineRepo}
}

func (h *BatchHandler) RegisterBatch(c *gin.Context) {
	var req dto.RegisterBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos: " + err.Error()})
		return
	}

	medicineID, err := uuid.Parse(req.MedicineID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medicine_id inválido"})
		return
	}

	batch, err := h.batchService.RegisterBatch(service.RegisterBatchInput{
		MedicineID: medicineID, Quantity: req.Quantity,
		ReceivedAt: req.ReceivedAt, ExpirationDate: req.ExpirationDate,
	})
	if err != nil {
		log.Printf("error registrando lote: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         batch.ID.String(),
		"batch_code": batch.BatchCode, // se lo devolvemos al frontend para que sepa qué código le tocó
		"message":    "lote registrado correctamente",
	})
}

func (h *BatchHandler) ListByMedicine(c *gin.Context) {
	medicineID, err := uuid.Parse(c.Param("medicineId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "medicine id inválido"})
		return
	}

	medicine, err := h.medicineService.FindByID(medicineID) // ver nota abajo, hay que agregar este método
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "medicamento no encontrado"})
		return
	}

	batches, err := h.batchService.ListByMedicine(medicineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los lotes"})
		return
	}

	response := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		status, days := service.BatchStatus(b.ExpirationDate, medicine.ExpirationAlertDays)
		response = append(response, dto.BatchResponse{
			ID:             b.ID.String(),
			MedicineID:     b.MedicineID.String(),
			MedicineName:   medicine.Name, // <- agregar esta línea
			BatchCode:      b.BatchCode,
			Quantity:       b.Quantity,
			ReceivedAt:     b.ReceivedAt.Format("2006-01-02"),
			ExpirationDate: b.ExpirationDate.Format("2006-01-02"),
			Status:         status,
			DaysRemaining:  days,
		})
	}
	c.JSON(http.StatusOK, response)
}

// ListExpiringSoon cubre RF-023. Acepta ?days=30 opcional; si no se
// especifica, usa 30 como valor por defecto.
func (h *BatchHandler) ListExpiringSoon(c *gin.Context) {
	days := 30
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil {
			days = parsed
		}
	}

	batches, err := h.batchService.ListExpiringSoon(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los lotes próximos a vencer"})
		return
	}
	c.JSON(http.StatusOK, toBatchResponses(batches, h.medicineRepo))
}

// ListExpired cubre RF-024.
func (h *BatchHandler) ListExpired(c *gin.Context) {
	batches, err := h.batchService.ListExpired()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudieron cargar los lotes vencidos"})
		return
	}
	c.JSON(http.StatusOK, toBatchResponses(batches, h.medicineRepo))
}

func toBatchResponses(batches []domain.Batch, medicineRepo repository.MedicineRepository) []dto.BatchResponse {
	response := make([]dto.BatchResponse, 0, len(batches))
	for _, b := range batches {
		alertDays := 30
		medicineName := ""

		if medicine, err := medicineRepo.FindByID(b.MedicineID); err == nil {
			alertDays = medicine.ExpirationAlertDays
			medicineName = medicine.Name
		}

		status, days := service.BatchStatus(b.ExpirationDate, alertDays)
		response = append(response, dto.BatchResponse{
			ID: b.ID.String(), MedicineID: b.MedicineID.String(),
			MedicineName: medicineName, // <- nuevo
			BatchCode:    b.BatchCode,
			Quantity:     b.Quantity, ReceivedAt: b.ReceivedAt.Format("2006-01-02"),
			ExpirationDate: b.ExpirationDate.Format("2006-01-02"),
			Status:         status, DaysRemaining: days,
		})
	}
	return response
}
