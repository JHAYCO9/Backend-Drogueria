package service

import (
	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

type RegisterIngresoInput struct {
	BatchID  uuid.UUID
	Quantity int
	UserID   uuid.UUID
}

type InventoryService interface {
	RegisterIngreso(input RegisterIngresoInput) error
}

type inventoryService struct {
	batchRepo    repository.BatchRepository
	movementRepo repository.InventoryMovementRepository
}

func NewInventoryService(batchRepo repository.BatchRepository, movementRepo repository.InventoryMovementRepository) InventoryService {
	return &inventoryService{batchRepo: batchRepo, movementRepo: movementRepo}
}

// RegisterIngreso cubre RF-016: suma cantidad a un lote existente y
// registra el movimiento en el historial. La actualización de existencias
// es automática porque el stock siempre se calcula sumando los lotes
// (no hay un campo "stock total" separado que se pueda desincronizar).
func (s *inventoryService) RegisterIngreso(input RegisterIngresoInput) error {
	batch, err := s.batchRepo.FindByID(input.BatchID)
	if err != nil {
		return ErrBatchNotFound
	}

	if err := s.batchRepo.IncreaseQuantity(batch.ID, input.Quantity); err != nil {
		return err
	}

	movement := &domain.InventoryMovement{
		ID:           uuid.New(),
		MedicineID:   batch.MedicineID,
		BatchID:      &batch.ID,
		MovementType: "ingreso",
		Quantity:     input.Quantity,
		UserID:       input.UserID,
	}
	return s.movementRepo.Create(movement)
}
