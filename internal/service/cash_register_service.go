package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

// ─────────────────────────────────────────────────────────────
// Errores de dominio del módulo de caja
// ─────────────────────────────────────────────────────────────

var ErrCashAlreadyOpen = errors.New("ya existe una caja abierta, debe cerrarse antes de abrir una nueva")
var ErrCashNotOpen = errors.New("no hay ninguna caja abierta actualmente")
var ErrCashNotOwnedByUser = errors.New("no tienes una caja abierta actualmente")
var ErrInvalidMovementAmount = errors.New("el monto del movimiento debe ser mayor a cero")
var ErrMovementReasonRequired = errors.New("la justificación del movimiento es obligatoria")

// ─────────────────────────────────────────────────────────────
// Tipos de salida del servicio
// ─────────────────────────────────────────────────────────────

// CashRegisterStatus es la respuesta resumida del estado actual (RF-049 existente).
type CashRegisterStatus struct {
	IsOpen          bool
	CashRegister    *domain.CashRegister
	OpenedByName    string
	ExpectedCash    float64 // solo efectivo: lo que debería contarse físicamente
	TotalNequiSales float64 // informativo: no se cuenta en el arqueo físico
	CurrentBalance  float64 // suma total (efectivo + nequi), solo informativo
}

// CashStatusDetail es la respuesta detallada del arqueo en tiempo real (RF-019).
type CashStatusDetail struct {
	CashRegister           *domain.CashRegister
	OpenedByName           string
	TotalVentasEfectivo    float64
	TotalVentasOtrosMedios float64
	TotalIngresosManual    float64
	TotalEgresosManual     float64
	SaldoTeorico           float64
	MovimientosManuales    []domain.CashMovement
}

// CashMovementFilterInput es el filtro del servicio para RF-051.
type CashMovementFilterInput struct {
	UserID *uuid.UUID
	From   *time.Time
	To     *time.Time
}

// ClosedRegisterDetail es el resultado enriquecido para RF-056.
type ClosedRegisterDetail struct {
	Register        domain.CashRegister
	OpenedByName    string
	ClosedByName    string
	TotalVentasCash float64
	TotalOtros      float64
	TotalIngresos   float64
	TotalEgresos    float64
	SaldoTeorico    float64
}

// ─────────────────────────────────────────────────────────────
// Interfaz del servicio
// ─────────────────────────────────────────────────────────────

type CashRegisterService interface {
	OpenCashRegister(userID uuid.UUID, openingAmount float64) (*domain.CashRegister, error)                        // RF-016 + RNF-008 + RNF-012
	GetStatus() (*CashRegisterStatus, error)                                                                       // RF-049 (existente)
	GetDetailedStatus() (*CashStatusDetail, error)                                                                 // RF-019
	CloseCashRegister(userID uuid.UUID, countedAmount float64, reason string) (*domain.CashRegister, error)        // RF-017 + RNF-011 + RNF-012
	RegisterManualMovement(userID uuid.UUID, movType, reason string, amount float64) (*domain.CashMovement, error) // RF-018 + RNF-010
	ListMovements(filter CashMovementFilterInput) ([]domain.CashMovement, error)                                   // RF-051
	ListClosedRegisters(from, to *time.Time, userID *uuid.UUID) ([]ClosedRegisterDetail, error)                    // RF-056
	// RequireCashOpenForUser es usado por el middleware RNF-009.
	RequireCashOpenForUser() (*domain.CashRegister, error)
}

// ─────────────────────────────────────────────────────────────
// Implementación
// ─────────────────────────────────────────────────────────────

type cashRegisterService struct {
	db              *gorm.DB
	cashRepo        repository.CashRegisterRepository
	movementRepo    repository.CashMovementRepository
	saleRepo        repository.SaleRepository
	userRepo        repository.UserRepository
	auditRepo       repository.AuditRepository
	reporteZService ReporteZService
}

func NewCashRegisterService(
	db *gorm.DB,
	cashRepo repository.CashRegisterRepository,
	movementRepo repository.CashMovementRepository,
	saleRepo repository.SaleRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditRepository,
	reporteZService ReporteZService,
) CashRegisterService {
	return &cashRegisterService{
		db: db, cashRepo: cashRepo, movementRepo: movementRepo, saleRepo: saleRepo,
		userRepo: userRepo, auditRepo: auditRepo,
		reporteZService: reporteZService,
	}
}

// NewCashRegisterServiceWithDB construye el servicio con acceso a la BD para
// transacciones atómicas (RNF-012). Úsalo en el router en lugar de NewCashRegisterService.
func NewCashRegisterServiceWithDB(
	db *gorm.DB,
	cashRepo repository.CashRegisterRepository,
	movementRepo repository.CashMovementRepository,
	saleRepo repository.SaleRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditRepository,
) CashRegisterService {
	return &cashRegisterService{db: db, cashRepo: cashRepo, movementRepo: movementRepo, saleRepo: saleRepo, userRepo: userRepo, auditRepo: auditRepo}
}

// ─────────────────────────────────────────────────────────────
// RF-016 + RNF-008 + RNF-012: Apertura de caja
// ─────────────────────────────────────────────────────────────

// OpenCashRegister verifica que no haya ninguna caja abierta (restricción GLOBAL),
// luego abre una nueva sesión en una transacción atómica.
// RNF-008: si ya existe una caja abierta (de cualquier usuario) → 400 Bad Request.
// RNF-012: verificación + creación dentro de tx.Begin().
func (s *cashRegisterService) OpenCashRegister(userID uuid.UUID, openingAmount float64) (*domain.CashRegister, error) {
	// Necesitamos tx para atomicidad (RNF-012)
	if s.db == nil {
		// fallback sin tx (compatibilidad con tests o uso sin DB inyectada)
		return s.openWithoutTx(userID, openingAmount)
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

	// RNF-008: verificar que no exista ninguna caja abierta (restricción GLOBAL).
	// Se hace dentro de la tx para evitar race conditions.
	if _, err := s.cashRepo.FindOpen(); err == nil {
		tx.Rollback()
		return nil, ErrCashAlreadyOpen
	}

	cr := &domain.CashRegister{
		ID:            uuid.New(),
		OpenedBy:      userID,
		OpeningAmount: openingAmount,
		Status:        "abierta",
		OpenedAt:      time.Now(),
	}

	if err := s.cashRepo.CreateTx(tx, cr); err != nil {
		tx.Rollback()
		return nil, err
	}

	// Registrar movimiento de apertura en la misma transacción.
	mov := &domain.CashMovement{
		ID: uuid.New(), CashRegisterID: cr.ID,
		MovementType: "apertura", Amount: openingAmount,
		Description: "Apertura de caja", MovementCategory: "auto", UserID: userID,
	}
	if err := s.movementRepo.CreateTx(tx, mov); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(userID, "cash_register_open", fmt.Sprintf("Apertura de caja ID: %s con monto inicial de %.2f", cr.ID.String(), openingAmount), "")
	}

	return cr, nil
}

func (s *cashRegisterService) openWithoutTx(userID uuid.UUID, openingAmount float64) (*domain.CashRegister, error) {
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
		Amount: openingAmount, Description: "Apertura de caja", MovementCategory: "auto", UserID: userID,
	})
	return cr, nil
}

// ─────────────────────────────────────────────────────────────
// RF-049: Estado resumido (mantiene compatibilidad con handler existente)
// ─────────────────────────────────────────────────────────────

func (s *cashRegisterService) GetStatus() (*CashRegisterStatus, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return &CashRegisterStatus{IsOpen: false}, nil
	}

	openedByName := ""
	if u, err := s.userRepo.FindByID(cr.OpenedBy); err == nil {
		openedByName = u.FullName
	}

	expectedCash, totalNequi, err := s.calculateBalancesLegacy(cr.ID, cr.OpeningAmount)
	if err != nil {
		return nil, err
	}

	return &CashRegisterStatus{
		IsOpen: true, CashRegister: cr, OpenedByName: openedByName,
		ExpectedCash: expectedCash, TotalNequiSales: totalNequi,
		CurrentBalance: expectedCash + totalNequi,
	}, nil
}

// ─────────────────────────────────────────────────────────────
// RF-019: Estado detallado del arqueo en tiempo real
// ─────────────────────────────────────────────────────────────

// GetDetailedStatus calcula el arqueo en tiempo real con todos los totales
// desglosados: ventas por método de pago, movimientos manuales, saldo teórico.
func (s *cashRegisterService) GetDetailedStatus() (*CashStatusDetail, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrCashNotOpen
	}

	openedByName := ""
	if u, err := s.userRepo.FindByID(cr.OpenedBy); err == nil {
		openedByName = u.FullName
	}

	// Ventas en efectivo de la sesión.
	totalEfectivo, err := s.saleRepo.SumCashSalesByCashRegister(cr.ID)
	if err != nil {
		return nil, err
	}

	// Ventas por otros medios (informativo).
	totalOtros, err := s.saleRepo.SumOtherSalesByCashRegister(cr.ID)
	if err != nil {
		return nil, err
	}

	// Total de ingresos manuales de la sesión.
	totalIngresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "ingreso_manual")
	if err != nil {
		return nil, err
	}

	// Total de egresos manuales de la sesión.
	totalEgresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "egreso_manual")
	if err != nil {
		return nil, err
	}

	// Fórmula del saldo teórico (RF-019):
	// SaldoTeorico = MontoInicial + VentasEfectivo + IngresosManual - EgresosManual
	saldo := cr.OpeningAmount + totalEfectivo + totalIngresos - totalEgresos

	// Lista de movimientos manuales de la sesión.
	movimientos, err := s.movementRepo.FindManualByCashRegister(cr.ID)
	if err != nil {
		return nil, err
	}

	return &CashStatusDetail{
		CashRegister:           cr,
		OpenedByName:           openedByName,
		TotalVentasEfectivo:    totalEfectivo,
		TotalVentasOtrosMedios: totalOtros,
		TotalIngresosManual:    totalIngresos,
		TotalEgresosManual:     totalEgresos,
		SaldoTeorico:           saldo,
		MovimientosManuales:    movimientos,
	}, nil
}

// ─────────────────────────────────────────────────────────────
// RF-017 + RNF-011 + RNF-012: Cierre de caja con arqueo
// ─────────────────────────────────────────────────────────────

// CloseCashRegister realiza el cierre de caja en una transacción atómica (RNF-012).
// Calcula el monto esperado con la fórmula completa y determina el tipo de discrepancia (RNF-011).
func (s *cashRegisterService) CloseCashRegister(userID uuid.UUID, countedAmount float64, reason string) (*domain.CashRegister, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrCashNotOpen
	}

	// ── Fórmula RF-017 ──────────────────────────────────────────────────────
	// MontoEsperado = MontoInicial + VentasEfectivo + IngresosManual - EgresosManual
	totalEfectivo, err := s.saleRepo.SumCashSalesByCashRegister(cr.ID)
	if err != nil {
		return nil, err
	}
	totalIngresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "ingreso_manual")
	if err != nil {
		return nil, err
	}
	totalEgresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "egreso_manual")
	if err != nil {
		return nil, err
	}

	expectedAmount := cr.OpeningAmount + totalEfectivo + totalIngresos - totalEgresos
	difference := countedAmount - expectedAmount

	// RNF-011: clasificación de la discrepancia (INALTERABLES una vez calculados).
	discrepancyType := discrepancyTypeOf(difference)

	now := time.Now()
	cr.Status = "cerrada"
	cr.ClosedBy = &userID
	cr.ClosedAt = &now
	cr.ExpectedAmount = &expectedAmount
	cr.CountedAmount = &countedAmount
	cr.DifferenceAmount = &difference
	cr.DiscrepancyType = &discrepancyType

	if difference != 0 && reason == "" {
		reason = "Sin motivo especificado"
	}
	if reason != "" {
		cr.DifferenceReason = &reason
	}

	// RNF-012: todo dentro de una transacción.
	if s.db != nil {
		return s.closeWithTx(userID, countedAmount, cr)
	}

	// fallback sin tx
	if err := s.cashRepo.Update(cr); err != nil {
		return nil, err
	}
	_ = s.movementRepo.Create(&domain.CashMovement{
		ID: uuid.New(), CashRegisterID: cr.ID, MovementType: "cierre",
		Amount: countedAmount, Description: "Cierre de caja (arqueo de efectivo)",
		MovementCategory: "auto", UserID: userID,
	})

	return cr, nil
}

func (s *cashRegisterService) closeWithTx(userID uuid.UUID, countedAmount float64, cr *domain.CashRegister) (*domain.CashRegister, error) {
	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := s.cashRepo.UpdateTx(tx, cr); err != nil {
		tx.Rollback()
		return nil, err
	}

	mov := &domain.CashMovement{
		ID: uuid.New(), CashRegisterID: cr.ID, MovementType: "cierre",
		Amount: countedAmount, Description: "Cierre de caja (arqueo de efectivo)",
		MovementCategory: "auto", UserID: userID,
	}
	if err := s.movementRepo.CreateTx(tx, mov); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		discrepancy := "NONE"
		if cr.DiscrepancyType != nil {
			discrepancy = *cr.DiscrepancyType
		}
		expected := 0.0
		if cr.ExpectedAmount != nil {
			expected = *cr.ExpectedAmount
		}
		diff := 0.0
		if cr.DifferenceAmount != nil {
			diff = *cr.DifferenceAmount
		}
		_ = s.auditRepo.Log(userID, "cash_register_close", fmt.Sprintf("Cierre de caja ID: %s. Esperado: %.2f, Contado: %.2f, Diferencia: %.2f (%s)", cr.ID.String(), expected, countedAmount, diff, discrepancy), "")
	}

	return cr, nil
}

// discrepancyTypeOf determina el tipo de discrepancia dado la diferencia calculada.
// RNF-011: este resultado es INALTERABLE una vez persistido.
func discrepancyTypeOf(difference float64) string {
	switch {
	case difference > 0:
		return "EXCESS"
	case difference < 0:
		return "DEFICIT"
	default:
		return "NONE"
	}
}

// ─────────────────────────────────────────────────────────────
// RF-018 + RNF-010: Registrar movimiento manual
// ─────────────────────────────────────────────────────────────

// RegisterManualMovement registra un ingreso o egreso manual de caja (RF-018).
// RNF-010: NO existe endpoint para actualizar ni eliminar estos registros.
// Los tipos válidos son "ingreso_manual" y "egreso_manual".
func (s *cashRegisterService) RegisterManualMovement(
	userID uuid.UUID, movType, reason string, amount float64,
) (*domain.CashMovement, error) {
	// Verificar que hay una caja abierta.
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrCashNotOpen
	}

	if amount <= 0 {
		return nil, ErrInvalidMovementAmount
	}
	if reason == "" {
		return nil, ErrMovementReasonRequired
	}

	movAmount := amount
	if movType == "egreso_manual" {
		movAmount = -amount
	}

	mov := &domain.CashMovement{
		ID:               uuid.New(),
		CashRegisterID:   cr.ID,
		MovementType:     "ajuste", // 'ajuste' está garantizado en la restricción original de la BD
		Amount:           movAmount,
		Description:      "Movimiento manual registrado por cajero",
		Reason:           reason,
		UserID:           userID,
		MovementCategory: "manual",
		CreatedAt:        time.Now(),
	}

	if err := s.movementRepo.Create(mov); err != nil {
		return nil, err
	}

	if s.auditRepo != nil {
		_ = s.auditRepo.Log(userID, "cash_movement_created", fmt.Sprintf("Movimiento manual de caja (%s) por %.2f. Motivo: %s", movType, amount, reason), "")
	}

	return mov, nil
}

// ─────────────────────────────────────────────────────────────
// RNF-009: Verificación de caja abierta (usada por el middleware)
// ─────────────────────────────────────────────────────────────

// RequireCashOpenForUser verifica que exista una caja abierta globalmente.
// Retorna la caja si está abierta o ErrCashNotOpen si no hay ninguna.
func (s *cashRegisterService) RequireCashOpenForUser() (*domain.CashRegister, error) {
	cr, err := s.cashRepo.FindOpen()
	if err != nil {
		return nil, ErrCashNotOpen
	}
	return cr, nil
}

// ─────────────────────────────────────────────────────────────
// RF-051: Listar movimientos (existente)
// ─────────────────────────────────────────────────────────────

func (s *cashRegisterService) ListMovements(filter CashMovementFilterInput) ([]domain.CashMovement, error) {
	return s.movementRepo.FindFiltered(repository.CashMovementFilter{
		UserID: filter.UserID, From: filter.From, To: filter.To,
	})
}

// ─────────────────────────────────────────────────────────────
// Helpers privados (compatibilidad con GetStatus existente)
// ─────────────────────────────────────────────────────────────

// calculateBalancesLegacy mantiene la lógica original basada en movimientos
// de tipo venta_efectivo/venta_nequi (RF-049 existente).
func (s *cashRegisterService) calculateBalancesLegacy(cashRegisterID uuid.UUID, openingAmount float64) (expectedCash float64, totalNequi float64, err error) {
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

// ─────────────────────────────────────────────────────────────
// RF-056: Historial de cierres de caja con filtros
// ─────────────────────────────────────────────────────────────

// ListClosedRegisters retorna el historial de cajas cerradas con totales calculados.
// Filtros: from/to por fecha de cierre, userID por quien cerró.
func (s *cashRegisterService) ListClosedRegisters(from, to *time.Time, userID *uuid.UUID) ([]ClosedRegisterDetail, error) {
	registers, err := s.cashRepo.FindClosed(from, to, userID)
	if err != nil {
		return nil, err
	}

	result := make([]ClosedRegisterDetail, 0, len(registers))
	for _, cr := range registers {
		detail := ClosedRegisterDetail{Register: cr}

		// Nombre del usuario que abrió la caja
		if u, err := s.userRepo.FindByID(cr.OpenedBy); err == nil {
			detail.OpenedByName = u.FullName
		}
		// Nombre del usuario que cerró la caja
		if cr.ClosedBy != nil {
			if u, err := s.userRepo.FindByID(*cr.ClosedBy); err == nil {
				detail.ClosedByName = u.FullName
			}
		}

		// Totales calculados para la sesión
		if cash, err := s.saleRepo.SumCashSalesByCashRegister(cr.ID); err == nil {
			detail.TotalVentasCash = cash
		}
		if otros, err := s.saleRepo.SumOtherSalesByCashRegister(cr.ID); err == nil {
			detail.TotalOtros = otros
		}
		if ingresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "ingreso_manual"); err == nil {
			detail.TotalIngresos = ingresos
		}
		if egresos, err := s.movementRepo.SumByTypeAndSession(cr.ID, "egreso_manual"); err == nil {
			detail.TotalEgresos = egresos
		}
		detail.SaldoTeorico = cr.OpeningAmount + detail.TotalVentasCash + detail.TotalIngresos - detail.TotalEgresos

		result = append(result, detail)
	}
	return result, nil
}
