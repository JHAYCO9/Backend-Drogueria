package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrCashAlreadyOpen = errors.New("ya existe una caja abierta, debe cerrarse antes de abrir una nueva")
var ErrCashNotOpen = errors.New("no hay ninguna caja abierta actualmente")

type CashRegisterStatus struct {
	IsOpen          bool
	CashRegister    *domain.CashRegister
	OpenedByName    string
	ExpectedCash    float64 // solo efectivo, lo que debería contarse físicamente
	TotalNequiSales float64 // informativo, no se cuenta en el arqueo
	CurrentBalance  float64 // suma total (efectivo + nequi), solo informativo
}

type CashMovementFilterInput struct {
	UserID *uuid.UUID
	From   *time.Time
	To     *time.Time
}

type CashRegisterService interface {
	OpenCashRegister(userID uuid.UUID, openingAmount float64) (*domain.CashRegister, error)                 // RF-048
	GetStatus() (*CashRegisterStatus, error)                                                                // RF-049
	CloseCashRegister(userID uuid.UUID, countedAmount float64, reason string) (*domain.CashRegister, error) // RF-052+053+054
	ListMovements(filter CashMovementFilterInput) ([]domain.CashMovement, error)                            // RF-051
}

type cashRegisterService struct {
	cashRepo     repository.CashRegisterRepository
	movementRepo repository.CashMovementRepository
	saleRepo     repository.SaleRepository
	userRepo     repository.UserRepository
}

func NewCashRegisterService(
	cashRepo repository.CashRegisterRepository,
	movementRepo repository.CashMovementRepository,
	saleRepo repository.SaleRepository,
	userRepo repository.UserRepository,
) CashRegisterService {
	return &cashRegisterService{cashRepo: cashRepo, movementRepo: movementRepo, saleRepo: saleRepo, userRepo: userRepo}
}

// OpenCashRegister cubre RF-048 + registra el movimiento "apertura" (RF-050).
func (s *cashRegisterService) OpenCashRegister(userID uuid.UUID, openingAmount float64) (*domain.CashRegister, error) {
	if _, err := s.cashRepo.FindOpen(); err == nil {
		return nil, ErrCashAlreadyOpen
	}

	cr := &domain.CashRegister{
		ID: uuid.New(), OpenedBy: userID, OpeningAmount: openingAmount,
		Status: "abierta", OpenedAt: time.Now(),
	}

	if err := s.cashRepo.Create(cr); err != nil {
		return nil, err
	}

	_ = s.movementRepo.Create(&domain.CashMovement{
		ID: uuid.New(), CashRegisterID: cr.ID, MovementType: "apertura",
		Amount: openingAmount, Description: "Apertura de caja", UserID: userID,
	})

	return cr, nil
}

func (s *cashRegisterService) GetStatus() (*CashRegisterStatus, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return &CashRegisterStatus{IsOpen: false}, nil
	}

	openedByName := ""
	if u, err := s.userRepo.FindByID(cr.OpenedBy); err == nil {
		openedByName = u.FullName
	}

	expectedCash, totalNequi, err := s.calculateBalances(cr.ID, cr.OpeningAmount)
	if err != nil {
		return nil, err
	}

	return &CashRegisterStatus{
		IsOpen: true, CashRegister: cr, OpenedByName: openedByName,
		ExpectedCash: expectedCash, TotalNequiSales: totalNequi,
		CurrentBalance: expectedCash + totalNequi,
	}, nil
}

// CloseCashRegister cubre RF-052 (arqueo: valor esperado vs contado),
// RF-053 (registra diferencia y motivo) y RF-054 (verifica apertura
// activa, calcula totales, marca hora de cierre y cambia el estado).
func (s *cashRegisterService) CloseCashRegister(userID uuid.UUID, countedAmount float64, reason string) (*domain.CashRegister, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrCashNotOpen
	}

	expectedCash, _, err := s.calculateBalances(cr.ID, cr.OpeningAmount)
	if err != nil {
		return nil, err
	}
	difference := countedAmount - expectedCash

	now := time.Now()
	cr.Status = "cerrada"
	cr.ClosedBy = &userID
	cr.ClosedAt = &now
	cr.ExpectedAmount = &expectedCash
	cr.CountedAmount = &countedAmount
	cr.DifferenceAmount = &difference

	if difference != 0 {
		if reason == "" {
			reason = "Sin motivo especificado"
		}
		cr.DifferenceReason = &reason
	}

	if err := s.cashRepo.Update(cr); err != nil {
		return nil, err
	}

	_ = s.movementRepo.Create(&domain.CashMovement{
		ID: uuid.New(), CashRegisterID: cr.ID, MovementType: "cierre",
		Amount: countedAmount, Description: "Cierre de caja (arqueo de efectivo)", UserID: userID,
	})

	return cr, nil
}

// ListMovements cubre RF-051: filtra por fecha y usuario.
func (s *cashRegisterService) ListMovements(filter CashMovementFilterInput) ([]domain.CashMovement, error) {
	return s.movementRepo.FindFiltered(repository.CashMovementFilter{
		UserID: filter.UserID, From: filter.From, To: filter.To,
	})
}

// calculateBalances devuelve por separado el efectivo esperado (lo que
// debería estar físicamente en la caja: valor inicial + ventas en
// efectivo) y el total vendido por Nequi (informativo, nunca se cuenta
// en el arqueo porque ese dinero nunca llega físico a la caja).
func (s *cashRegisterService) calculateBalances(cashRegisterID uuid.UUID, openingAmount float64) (expectedCash float64, totalNequi float64, err error) {
	movements, err := s.movementRepo.FindByCashRegister(cashRegisterID)
	if err != nil {
		return openingAmount, 0, err
	}

	expectedCash = openingAmount
	for _, m := range movements {
		switch m.MovementType {
		case "venta_efectivo":
			expectedCash += m.Amount
		case "venta_nequi":
			totalNequi += m.Amount
		}
	}
	return expectedCash, totalNequi, nil
}
