package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrInsufficientStock = errors.New("stock insuficiente para completar la operacion")
var ErrBatchBelongsToOtherMedicine = errors.New("el lote no pertenece al medicamento indicado")

type RegisterIngresoInput struct {
	BatchID  uuid.UUID
	Quantity int
	UserID   uuid.UUID
}

type RegisterAdjustmentInput struct {
	MedicineID  uuid.UUID
	BatchID     uuid.UUID
	NewQuantity int
	Reason      string
	UserID      uuid.UUID
}

type MovementFilters struct {
	MovementType string
	From         *time.Time
	To           *time.Time
}

// FEFODeductionResult muestra el detalle de cada lote que fue afectado
// durante un descuento FEFO.
type FEFODeductionResult struct {
	BatchID    uuid.UUID
	BatchCode  string
	Deducted   int
	Remaining  int
}

type InventoryService interface {
	RegisterIngreso(input RegisterIngresoInput) error
	RegisterAdjustment(input RegisterAdjustmentInput) error
	ListMovements(filters MovementFilters) ([]domain.InventoryMovement, error)
	GetStockHistory(from, to *time.Time, movementType string) ([]domain.InventoryMovement, error)
	GetGeneralInventory() ([]GeneralInventoryItem, error)
	GetCriticalInventory(expirationDays int) ([]CriticalInventoryItem, error)
	DeductStockFEFO(medicineID uuid.UUID, quantity int, userID uuid.UUID) ([]FEFODeductionResult, error)
}

// GeneralInventoryItem es el consolidado de RF-028.
type GeneralInventoryItem struct {
	Medicine         domain.Medicine
	TotalStock       int
	NextExpiration   *time.Time
	AvailableBatches int
}

// CriticalInventoryItem es el consolidado de RF-029.
type CriticalInventoryItem struct {
	Medicine   domain.Medicine
	TotalStock int
	Reasons    []string
}

type inventoryService struct {
	batchRepo      repository.BatchRepository
	movementRepo   repository.InventoryMovementRepository
	adjustmentRepo repository.InventoryAdjustmentRepository
	medicineRepo   repository.MedicineRepository
	userRepo       repository.UserRepository
}

func NewInventoryService(
	batchRepo repository.BatchRepository,
	movementRepo repository.InventoryMovementRepository,
	adjustmentRepo repository.InventoryAdjustmentRepository,
	medicineRepo repository.MedicineRepository,
	userRepo repository.UserRepository,
) InventoryService {
	return &inventoryService{
		batchRepo:      batchRepo,
		movementRepo:   movementRepo,
		adjustmentRepo: adjustmentRepo,
		medicineRepo:   medicineRepo,
		userRepo:       userRepo,
	}
}

// RegisterIngreso cubre RF-016: suma cantidad a un lote existente y
// registra el movimiento en el historial.
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

// RegisterAdjustment cubre RF-017: registra un ajuste manual de inventario.
// Guarda la cantidad anterior y nueva, actualiza el lote, y crea un
// movimiento en el historial.
func (s *inventoryService) RegisterAdjustment(input RegisterAdjustmentInput) error {
	batch, err := s.batchRepo.FindByID(input.BatchID)
	if err != nil {
		return ErrBatchNotFound
	}

	if batch.MedicineID != input.MedicineID {
		return ErrBatchBelongsToOtherMedicine
	}

	previousQty := batch.Quantity

	// Registrar el ajuste con el detalle completo
	adjustment := &domain.InventoryAdjustment{
		ID:               uuid.New(),
		MedicineID:       input.MedicineID,
		BatchID:          input.BatchID,
		PreviousQuantity: previousQty,
		NewQuantity:      input.NewQuantity,
		Reason:           input.Reason,
		UserID:           input.UserID,
	}
	if err := s.adjustmentRepo.Create(adjustment); err != nil {
		return err
	}

	// Actualizar la cantidad del lote inmediatamente
	if err := s.batchRepo.UpdateQuantity(input.BatchID, input.NewQuantity); err != nil {
		return err
	}

	// Registrar en el historial unificado de movimientos
	diff := input.NewQuantity - previousQty
	movement := &domain.InventoryMovement{
		ID:           uuid.New(),
		MedicineID:   input.MedicineID,
		BatchID:      &input.BatchID,
		MovementType: "ajuste",
		Quantity:     diff,
		Reason:       fmt.Sprintf("%s (anterior: %d, nuevo: %d)", input.Reason, previousQty, input.NewQuantity),
		UserID:       input.UserID,
		ReferenceID:  &adjustment.ID,
	}
	return s.movementRepo.Create(movement)
}

// ListMovements cubre RF-019: retorna el historial de movimientos
// enriquecido con nombres de medicamento, lote y usuario.
func (s *inventoryService) ListMovements(filters MovementFilters) ([]domain.InventoryMovement, error) {
	movements, err := s.movementRepo.FindAll(filters.MovementType, filters.From, filters.To)
	if err != nil {
		return nil, err
	}

	// Cachear nombres para evitar consultas repetidas
	medicineCache := make(map[string]string)
	batchCache := make(map[string]string)
	userCache := make(map[string]string)

	for i := range movements {
		// Nombre del medicamento
		medID := movements[i].MedicineID.String()
		if name, ok := medicineCache[medID]; ok {
			movements[i].MedicineName = name
		} else {
			if med, err := s.medicineRepo.FindByID(movements[i].MedicineID); err == nil {
				medicineCache[medID] = med.Name
				movements[i].MedicineName = med.Name
			}
		}

		// Codigo del lote
		if movements[i].BatchID != nil {
			batchID := movements[i].BatchID.String()
			if code, ok := batchCache[batchID]; ok {
				movements[i].BatchCode = code
			} else {
				if batch, err := s.batchRepo.FindByID(*movements[i].BatchID); err == nil {
					batchCache[batchID] = batch.BatchCode
					movements[i].BatchCode = batch.BatchCode
				}
			}
		}

		// Username
		userID := movements[i].UserID.String()
		if username, ok := userCache[userID]; ok {
			movements[i].Username = username
		} else {
			if user, err := s.userRepo.FindByID(movements[i].UserID); err == nil {
				userCache[userID] = user.Username
				movements[i].Username = user.Username
			} else {
				userCache[userID] = "usuario eliminado"
				movements[i].Username = "usuario eliminado"
			}
		}
	}

	return movements, nil
}

// GetStockHistory cubre RF-030: historial de existencias filtrado por
// rango de fechas y tipo de movimiento.
func (s *inventoryService) GetStockHistory(from, to *time.Time, movementType string) ([]domain.InventoryMovement, error) {
	movements, err := s.movementRepo.FindByDateRange(from, to, movementType)
	if err != nil {
		return nil, err
	}

	// Enriquecer con los mismos datos que ListMovements
	medicineCache := make(map[string]string)
	batchCache := make(map[string]string)
	userCache := make(map[string]string)

	for i := range movements {
		medID := movements[i].MedicineID.String()
		if name, ok := medicineCache[medID]; ok {
			movements[i].MedicineName = name
		} else {
			if med, err := s.medicineRepo.FindByID(movements[i].MedicineID); err == nil {
				medicineCache[medID] = med.Name
				movements[i].MedicineName = med.Name
			}
		}

		if movements[i].BatchID != nil {
			batchID := movements[i].BatchID.String()
			if code, ok := batchCache[batchID]; ok {
				movements[i].BatchCode = code
			} else {
				if batch, err := s.batchRepo.FindByID(*movements[i].BatchID); err == nil {
					batchCache[batchID] = batch.BatchCode
					movements[i].BatchCode = batch.BatchCode
				}
			}
		}

		userID := movements[i].UserID.String()
		if username, ok := userCache[userID]; ok {
			movements[i].Username = username
		} else {
			if user, err := s.userRepo.FindByID(movements[i].UserID); err == nil {
				userCache[userID] = user.Username
				movements[i].Username = user.Username
			} else {
				userCache[userID] = "usuario eliminado"
				movements[i].Username = "usuario eliminado"
			}
		}
	}

	return movements, nil
}

// GetGeneralInventory cubre RF-028: consolidado de inventario por
// medicamento con stock total, proximo vencimiento y numero de lotes.
func (s *inventoryService) GetGeneralInventory() ([]GeneralInventoryItem, error) {
	medicines, err := s.medicineRepo.FindAllIncludingInactive()
	if err != nil {
		return nil, err
	}

	result := make([]GeneralInventoryItem, 0, len(medicines))
	for _, m := range medicines {
		stock, err := s.batchRepo.SumStockByMedicine(m.ID)
		if err != nil {
			stock = 0
		}

		batchCount, err := s.batchRepo.CountAvailableByMedicine(m.ID)
		if err != nil {
			batchCount = 0
		}

		var nextExp *time.Time
		if earliest, err := s.batchRepo.FindEarliestExpiration(m.ID); err == nil {
			nextExp = &earliest.ExpirationDate
		}

		result = append(result, GeneralInventoryItem{
			Medicine:         m,
			TotalStock:       stock,
			NextExpiration:   nextExp,
			AvailableBatches: batchCount,
		})
	}

	return result, nil
}

// GetCriticalInventory cubre RF-029: medicamentos que cumplen al menos
// una condicion critica:
//   1. Proximos a vencer (dentro del rango de dias parametrizado)
//   2. Stock bajo (stock actual <= stock minimo)
//   3. Completamente vencidos
func (s *inventoryService) GetCriticalInventory(expirationDays int) ([]CriticalInventoryItem, error) {
	medicines, err := s.medicineRepo.FindAll()
	if err != nil {
		return nil, err
	}

	if expirationDays <= 0 {
		expirationDays = 30
	}

	var result []CriticalInventoryItem
	now := time.Now()
	limit := now.AddDate(0, 0, expirationDays)

	for _, m := range medicines {
		stock, err := s.batchRepo.SumStockByMedicine(m.ID)
		if err != nil {
			continue
		}

		var reasons []string

		// Condicion 1: stock bajo
		if stock <= m.MinStock {
			reasons = append(reasons, fmt.Sprintf("stock bajo (%d/%d)", stock, m.MinStock))
		}

		// Verificar lotes del medicamento
		batches, err := s.batchRepo.FindByMedicine(m.ID)
		if err != nil {
			continue
		}

		hasExpiringSoon := false
		hasExpired := false
		for _, b := range batches {
			if b.Quantity <= 0 {
				continue
			}
			if b.ExpirationDate.Before(now) || b.ExpirationDate.Equal(now) {
				hasExpired = true
			} else if b.ExpirationDate.Before(limit) || b.ExpirationDate.Equal(limit) {
				hasExpiringSoon = true
			}
		}

		// Condicion 2: proximo a vencer
		if hasExpiringSoon {
			reasons = append(reasons, "tiene lotes proximos a vencer")
		}

		// Condicion 3: lotes vencidos
		if hasExpired {
			reasons = append(reasons, "tiene lotes vencidos")
		}

		if len(reasons) > 0 {
			result = append(result, CriticalInventoryItem{
				Medicine:   m,
				TotalStock: stock,
				Reasons:    reasons,
			})
		}
	}

	return result, nil
}

// DeductStockFEFO cubre RF-025: descuenta stock siguiendo la estrategia
// FEFO (First Expired, First Out). Selecciona automaticamente el lote
// con vencimiento mas proximo y, si la cantidad solicitada supera las
// existencias de ese lote, continua con el siguiente.
func (s *inventoryService) DeductStockFEFO(medicineID uuid.UUID, quantity int, userID uuid.UUID) ([]FEFODeductionResult, error) {
	if quantity <= 0 {
		return nil, errors.New("la cantidad a descontar debe ser mayor a cero")
	}

	batches, err := s.batchRepo.FindAvailableByMedicineFEFO(medicineID)
	if err != nil {
		return nil, err
	}

	// Verificar que hay stock suficiente en total
	totalAvailable := 0
	for _, b := range batches {
		totalAvailable += b.Quantity
	}
	if totalAvailable < quantity {
		return nil, ErrInsufficientStock
	}

	remaining := quantity
	var results []FEFODeductionResult

	for _, batch := range batches {
		if remaining <= 0 {
			break
		}

		deduct := batch.Quantity
		if deduct > remaining {
			deduct = remaining
		}

		if err := s.batchRepo.DecreaseQuantity(batch.ID, deduct); err != nil {
			return nil, err
		}

		// Registrar el movimiento de salida por venta
		batchID := batch.ID
		movement := &domain.InventoryMovement{
			ID:           uuid.New(),
			MedicineID:   medicineID,
			BatchID:      &batchID,
			MovementType: "venta",
			Quantity:     -deduct,
			Reason:       fmt.Sprintf("descuento FEFO del lote %s", batch.BatchCode),
			UserID:       userID,
		}
		if err := s.movementRepo.Create(movement); err != nil {
			return nil, err
		}

		results = append(results, FEFODeductionResult{
			BatchID:   batch.ID,
			BatchCode: batch.BatchCode,
			Deducted:  deduct,
			Remaining: batch.Quantity - deduct,
		})

		remaining -= deduct
	}

	return results, nil
}
