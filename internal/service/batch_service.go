package service

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrBatchNotFound = errors.New("lote no encontrado")

const dateLayout = "2006-01-02"

type RegisterBatchInput struct {
	MedicineID     uuid.UUID
	Quantity       int
	ReceivedAt     string
	ExpirationDate string
}

type BatchService interface {
	RegisterBatch(input RegisterBatchInput) (*domain.Batch, error)
	ListByMedicine(medicineID uuid.UUID) ([]domain.Batch, error)
	ListExpiringSoon(withinDays int) ([]domain.Batch, error) // RF-023
	ListExpired() ([]domain.Batch, error)                    // RF-024
	IsExpired(batchID uuid.UUID) (bool, error)               // usado por Ventas en Sprint 3
}

type batchService struct {
	batchRepo    repository.BatchRepository
	medicineRepo repository.MedicineRepository
}

func NewBatchService(batchRepo repository.BatchRepository, medicineRepo repository.MedicineRepository) BatchService {
	return &batchService{batchRepo: batchRepo, medicineRepo: medicineRepo}
}

func (s *batchService) RegisterBatch(input RegisterBatchInput) (*domain.Batch, error) {
	if _, err := s.medicineRepo.FindByID(input.MedicineID); err != nil {
		return nil, ErrMedicineNotFound
	}

	receivedAt, err := time.Parse(dateLayout, input.ReceivedAt)
	if err != nil {
		return nil, errors.New("fecha de ingreso inválida, formato esperado AAAA-MM-DD")
	}
	expirationDate, err := time.Parse(dateLayout, input.ExpirationDate)
	if err != nil {
		return nil, errors.New("fecha de vencimiento inválida, formato esperado AAAA-MM-DD")
	}

	batch := &domain.Batch{
		ID:             uuid.New(),
		MedicineID:     input.MedicineID,
		BatchCode:      generateBatchCode(receivedAt),
		Quantity:       input.Quantity,
		ReceivedAt:     receivedAt,
		ExpirationDate: expirationDate,
	}

	if err := s.batchRepo.Create(batch); err != nil {
		return nil, err
	}
	return batch, nil
}

// generateBatchCode arma un código legible tipo "LOTE-20260710-A3F2".
// Con una sola caja/administrador registrando lotes, el riesgo de choque
// entre dos códigos generados al mismo segundo es prácticamente nulo,
// pero igual usamos una tabla UNIQUE si algún día se necesita blindar más.
func generateBatchCode(receivedAt time.Time) string {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // sin caracteres ambiguos (0/O, 1/I)
	suffix := make([]byte, 4)
	for i := range suffix {
		suffix[i] = chars[rand.Intn(len(chars))]
	}
	return fmt.Sprintf("LOTE-%s-%s", receivedAt.Format("20060102"), strings.ToUpper(string(suffix)))
}

// ListByMedicine cubre RF-018.
func (s *batchService) ListByMedicine(medicineID uuid.UUID) ([]domain.Batch, error) {
	return s.batchRepo.FindByMedicine(medicineID)
}

func (s *batchService) ListExpiringSoon(withinDays int) ([]domain.Batch, error) {
	return s.batchRepo.FindExpiringSoon(withinDays)
}

func (s *batchService) ListExpired() ([]domain.Batch, error) {
	return s.batchRepo.FindExpired()
}

func (s *batchService) IsExpired(batchID uuid.UUID) (bool, error) {
	batch, err := s.batchRepo.FindByID(batchID)
	if err != nil {
		return false, ErrBatchNotFound
	}
	return batch.ExpirationDate.Before(time.Now()), nil
}

// BatchStatus calcula el estado de un lote para RF-018 (vigente / próximo
// a vencer / vencido), usando el umbral de días configurado en el
// medicamento (RF-022).
func BatchStatus(expirationDate time.Time, alertDays int) (status string, daysRemaining int) {
	daysRemaining = int(time.Until(expirationDate).Hours() / 24)
	switch {
	case daysRemaining < 0:
		return "vencido", daysRemaining
	case daysRemaining <= alertDays:
		return "proximo_a_vencer", daysRemaining
	default:
		return "vigente", daysRemaining
	}
}
