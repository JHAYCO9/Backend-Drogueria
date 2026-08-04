package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrReturnExceedsQuantity = errors.New("la cantidad a devolver supera lo vendido (o lo ya devuelto) para este producto")
var ErrSaleItemNotBelongToSale = errors.New("el producto indicado no pertenece a la venta seleccionada")
var ErrReturnNotFound = errors.New("la devolución especificada no fue encontrada")
var ErrReturnAlreadyVoided = errors.New("esta devolución ya ha sido anulada previamente")

type ReturnService interface {
	RegisterReturn(saleID, saleItemID, userID uuid.UUID, quantity int, reason string) (*domain.Return, error) // RF-089+091+095+RNF-028
	VoidReturn(returnID, userID uuid.UUID, reason string) (*domain.Return, error)                             // RF-094
	ListReturns() ([]ReturnDetail, error)                                                                     // RF-093
}

type ReturnDetail struct {
	Return       domain.Return
	SaleNumber   int
	MedicineName string
	Username     string
	VoidedByName string
}

type returnService struct {
	db           *gorm.DB
	returnRepo   repository.ReturnRepository
	saleRepo     repository.SaleRepository
	batchRepo    repository.BatchRepository
	movementRepo repository.InventoryMovementRepository
	medicineRepo repository.MedicineRepository
	userRepo     repository.UserRepository
	auditRepo    repository.AuditRepository
	cashRepo     repository.CashRegisterRepository // RNF-009: validar caja abierta
}

func NewReturnService(
	db *gorm.DB, returnRepo repository.ReturnRepository, saleRepo repository.SaleRepository,
	batchRepo repository.BatchRepository, movementRepo repository.InventoryMovementRepository,
	medicineRepo repository.MedicineRepository, userRepo repository.UserRepository,
	auditRepo repository.AuditRepository, cashRepo repository.CashRegisterRepository,
) ReturnService {
	return &returnService{
		db: db, returnRepo: returnRepo, saleRepo: saleRepo, batchRepo: batchRepo,
		movementRepo: movementRepo, medicineRepo: medicineRepo, userRepo: userRepo,
		auditRepo: auditRepo, cashRepo: cashRepo,
	}
}

// RegisterReturn cubre RF-089 (registrar devolución), RF-091 (reintegrar
// existencias al lote correspondiente y registrar el movimiento), RF-095
// (auditoría) y RNF-028 (todo dentro de una sola transacción atómica).
// RNF-009: valida que exista una caja abierta antes de permitir la devolución.
func (s *returnService) RegisterReturn(saleID, saleItemID, userID uuid.UUID, quantity int, reason string) (*domain.Return, error) {
	// RNF-009: no se puede registrar una devolución sin caja abierta.
	if _, err := s.cashRepo.FindOpen(); err != nil {
		return nil, ErrNoCashOpen
	}

	sale, err := s.saleRepo.FindByID(saleID)
	if err != nil {
		return nil, ErrSaleNotFound
	}

	item, err := s.saleRepo.FindItemByID(saleItemID)
	if err != nil || item.SaleID != saleID {
		return nil, ErrSaleItemNotBelongToSale
	}

	alreadyReturned, err := s.returnRepo.SumReturnedForSaleItem(saleItemID)
	if err != nil {
		return nil, err
	}
	if alreadyReturned+quantity > item.UnitsConsumed {
		return nil, ErrReturnExceedsQuantity
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	ret := &domain.Return{
		ID: uuid.New(), SaleID: saleID, SaleItemID: saleItemID,
		MedicineID: item.MedicineID, BatchID: item.BatchID, Quantity: quantity,
		Reason: reason, UserID: userID, CreatedAt: time.Now(),
	}
	if err := s.returnRepo.CreateTx(tx, ret); err != nil {
		tx.Rollback()
		return nil, err
	}

	// RF-091: reintegra al mismo lote de donde salió originalmente.
	if err := tx.Model(&domain.Batch{}).Where("id = ?", item.BatchID).
		UpdateColumn("quantity", gorm.Expr("quantity + ?", quantity)).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	movement := &domain.InventoryMovement{
		ID: uuid.New(), MedicineID: item.MedicineID, BatchID: &item.BatchID,
		MovementType: "devolucion", Quantity: quantity, Reason: reason,
		UserID: userID, ReferenceID: &ret.ID,
	}
	if err := tx.Create(movement).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// RF-095: auditoría (fuera de la transacción, no crítico si falla)
	medicineName := ""
	if m, err := s.medicineRepo.FindByID(item.MedicineID); err == nil {
		medicineName = m.Name
	}
	detail := fmt.Sprintf("devolución de %d unidad(es) de %s de la venta #%d. Motivo: %s",
		quantity, medicineName, sale.SaleNumber, reason)
	_ = s.auditRepo.Log(userID, "return_created", detail, "")

	return ret, nil
}

// VoidReturn cubre RF-094: anular una devolución registrada por error por el administrador.
// Criterios de aceptación:
// 1. Motivo registrado.
// 2. Revertir los cambios en inventario (descontar del stock del lote).
// 3. Registrar en la auditoría.
// RNF-027: El registro NO se elimina del historial (permanencia garantizada).
func (s *returnService) VoidReturn(returnID, userID uuid.UUID, reason string) (*domain.Return, error) {
	ret, err := s.returnRepo.FindByID(returnID)
	if err != nil {
		return nil, ErrReturnNotFound
	}
	if strings.HasPrefix(ret.Reason, "[ANULADA]") || ret.IsVoided {
		return nil, ErrReturnAlreadyVoided
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	now := time.Now()
	ret.Reason = fmt.Sprintf("[ANULADA] %s (Motivo: %s)", ret.Reason, reason)
	ret.IsVoided = true
	ret.VoidedReason = reason
	ret.VoidedAt = &now
	ret.VoidedBy = &userID

	if err := s.returnRepo.UpdateTx(tx, ret); err != nil {
		tx.Rollback()
		return nil, err
	}

	// Revertir inventario: restar la cantidad devuelta previamente del lote correspondiente
	if err := tx.Model(&domain.Batch{}).Where("id = ? AND quantity >= ?", ret.BatchID, ret.Quantity).
		UpdateColumn("quantity", gorm.Expr("quantity - ?", ret.Quantity)).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("no se pudo revertir el stock del lote: %w", err)
	}

	movement := &domain.InventoryMovement{
		ID: uuid.New(), MedicineID: ret.MedicineID, BatchID: &ret.BatchID,
		MovementType: "anulacion_devolucion", Quantity: ret.Quantity, Reason: reason,
		UserID: userID, ReferenceID: &ret.ID,
	}
	if err := tx.Create(movement).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Auditoría
	medicineName := ""
	if m, err := s.medicineRepo.FindByID(ret.MedicineID); err == nil {
		medicineName = m.Name
	}
	detail := fmt.Sprintf("anulación de devolución #%s (%d unidad(es) de %s). Motivo: %s",
		ret.ID.String(), ret.Quantity, medicineName, reason)
	_ = s.auditRepo.Log(userID, "return_voided", detail, "")

	return ret, nil
}

// ListReturns cubre RF-093 + RNF-027.
func (s *returnService) ListReturns() ([]ReturnDetail, error) {
	returns, err := s.returnRepo.FindAll()
	if err != nil {
		return nil, err
	}

	details := make([]ReturnDetail, 0, len(returns))
	for _, r := range returns {
		if strings.HasPrefix(r.Reason, "[ANULADA]") {
			r.IsVoided = true
			r.VoidedReason = r.Reason
		}
		saleNumber := 0
		if sale, err := s.saleRepo.FindByID(r.SaleID); err == nil {
			saleNumber = sale.SaleNumber
		}
		medicineName := ""
		if m, err := s.medicineRepo.FindByID(r.MedicineID); err == nil {
			medicineName = m.Name
		}
		username := ""
		if u, err := s.userRepo.FindByID(r.UserID); err == nil {
			username = u.FullName
		}
		voidedByName := ""
		if r.VoidedBy != nil {
			if u, err := s.userRepo.FindByID(*r.VoidedBy); err == nil {
				voidedByName = u.FullName
			}
		}
		details = append(details, ReturnDetail{
			Return: r, SaleNumber: saleNumber, MedicineName: medicineName,
			Username: username, VoidedByName: voidedByName,
		})
	}
	return details, nil
}
