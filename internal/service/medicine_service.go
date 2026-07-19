package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrBarcodeTaken = errors.New("ya existe un medicamento registrado con ese codigo de barras")
var ErrMedicineNotFound = errors.New("medicamento no encontrado")
var ErrMedicineAlreadyInactive = errors.New("el medicamento ya se encuentra desactivado")
var ErrMedicineAlreadyActive = errors.New("el medicamento ya se encuentra activo")

type RegisterMedicineInput struct {
	Barcode       string
	Name          string
	Laboratory    string
	Concentration string
	Presentation  string
	PricePerPill  float64
	PricePerSeal  float64
	PricePerBox   float64
	MinStock      int
}

type UpdateMedicineInput struct {
	Name          *string
	Presentation  *string
	Concentration *string
	PricePerPill  *float64
	PricePerSeal  *float64
	PricePerBox   *float64
	MinStock      *int
}

type MedicineSearchParams struct {
	Name       string
	Barcode    string
	Laboratory string
	MinPrice   *float64
	MaxPrice   *float64
	HasStock   *bool
}

type CriticalMedicine struct {
	Medicine     domain.Medicine
	CurrentStock int
}

type MedicineService interface {
	RegisterMedicine(input RegisterMedicineInput) (*domain.Medicine, error)
	FindByBarcode(barcode string) (*domain.Medicine, error)
	ListMedicines() ([]domain.Medicine, error)
	UpdateMinStock(id uuid.UUID, minStock int) error
	ListCriticalMedicines() ([]CriticalMedicine, error)
	UpdateAlertDays(id uuid.UUID, days int) error
	FindByID(id uuid.UUID) (*domain.Medicine, error)
	UpdateMedicine(id uuid.UUID, input UpdateMedicineInput, userID uuid.UUID, ip string) (*domain.Medicine, error)
	DeactivateMedicine(id uuid.UUID, userID uuid.UUID, ip string) error
	ActivateMedicine(id uuid.UUID, userID uuid.UUID, ip string) error
	SearchMedicines(params MedicineSearchParams) ([]domain.Medicine, error)
}

type medicineService struct {
	medicineRepo repository.MedicineRepository
	batchRepo    repository.BatchRepository
	auditRepo    repository.AuditRepository
}

func (s *medicineService) FindByID(id uuid.UUID) (*domain.Medicine, error) {
	return s.medicineRepo.FindByID(id)
}

func NewMedicineService(medicineRepo repository.MedicineRepository, batchRepo repository.BatchRepository, auditRepo repository.AuditRepository) MedicineService {
	return &medicineService{medicineRepo: medicineRepo, batchRepo: batchRepo, auditRepo: auditRepo}
}

// RegisterMedicine cubre RF-011 completo: valida campos obligatorios
// (a nivel de DTO con `binding`), impide codigos de barras duplicados,
// y registra fecha de creacion (automatica via GORM). Incluye MinStock
// (RF-020) como parte del mismo registro inicial.
func (s *medicineService) RegisterMedicine(input RegisterMedicineInput) (*domain.Medicine, error) {
	exists, err := s.medicineRepo.ExistsByBarcode(input.Barcode)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBarcodeTaken
	}

	medicine := &domain.Medicine{
		ID:            uuid.New(),
		Barcode:       input.Barcode,
		Name:          input.Name,
		Laboratory:    input.Laboratory,
		Concentration: input.Concentration,
		Presentation:  input.Presentation,
		PricePerPill:  input.PricePerPill,
		PricePerSeal:  input.PricePerSeal,
		PricePerBox:   input.PricePerBox,
		MinStock:      input.MinStock,
		IsActive:      true,
		CreatedAt:     time.Now(),
	}

	if err := s.medicineRepo.Create(medicine); err != nil {
		return nil, err
	}
	return medicine, nil
}

// FindByBarcode es RF-027: recibe el codigo que capturo el lector fisico
// y devuelve el medicamento correspondiente.
func (s *medicineService) FindByBarcode(barcode string) (*domain.Medicine, error) {
	medicine, err := s.medicineRepo.FindByBarcode(barcode)
	if err != nil {
		return nil, ErrMedicineNotFound
	}
	return medicine, nil
}

func (s *medicineService) ListMedicines() ([]domain.Medicine, error) {
	return s.medicineRepo.FindAll()
}

// UpdateMinStock cubre RF-020 como endpoint independiente.
func (s *medicineService) UpdateMinStock(id uuid.UUID, minStock int) error {
	if minStock < 0 {
		return errors.New("el stock minimo debe ser mayor o igual a cero")
	}

	medicine, err := s.medicineRepo.FindByID(id)
	if err != nil {
		return ErrMedicineNotFound
	}

	medicine.MinStock = minStock
	return s.medicineRepo.Update(medicine)
}

// ListCriticalMedicines cubre RF-021: medicamentos cuyo stock actual
// es menor o igual a su stock minimo configurado.
func (s *medicineService) ListCriticalMedicines() ([]CriticalMedicine, error) {
	medicines, err := s.medicineRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var critical []CriticalMedicine
	for _, m := range medicines {
		stock, err := s.batchRepo.SumStockByMedicine(m.ID)
		if err != nil {
			continue
		}
		if stock <= m.MinStock {
			critical = append(critical, CriticalMedicine{Medicine: m, CurrentStock: stock})
		}
	}
	return critical, nil
}

func (s *medicineService) UpdateAlertDays(id uuid.UUID, days int) error {
	if _, err := s.medicineRepo.FindByID(id); err != nil {
		return ErrMedicineNotFound
	}
	return s.medicineRepo.UpdateAlertDays(id, days)
}

// UpdateMedicine cubre RF-012: modifica la informacion de un medicamento
// y registra la accion en el sistema de auditoria.
func (s *medicineService) UpdateMedicine(id uuid.UUID, input UpdateMedicineInput, userID uuid.UUID, ip string) (*domain.Medicine, error) {
	medicine, err := s.medicineRepo.FindByID(id)
	if err != nil {
		return nil, ErrMedicineNotFound
	}

	if input.Name != nil && *input.Name != "" {
		medicine.Name = *input.Name
	}
	if input.Presentation != nil {
		medicine.Presentation = *input.Presentation
	}
	if input.Concentration != nil {
		medicine.Concentration = *input.Concentration
	}
	if input.PricePerPill != nil {
		medicine.PricePerPill = *input.PricePerPill
	}
	if input.PricePerSeal != nil {
		medicine.PricePerSeal = *input.PricePerSeal
	}
	if input.PricePerBox != nil {
		medicine.PricePerBox = *input.PricePerBox
	}
	if input.MinStock != nil {
		medicine.MinStock = *input.MinStock
	}

	if err := s.medicineRepo.Update(medicine); err != nil {
		return nil, err
	}

	// Registrar en auditoria (RF-012 criterio de aceptacion)
	detail := fmt.Sprintf("medicamento modificado: %s (ID: %s)", medicine.Name, medicine.ID.String())
	_ = s.auditRepo.Log(userID, "medicine_update", detail, ip)

	return medicine, nil
}

// DeactivateMedicine cubre RF-013: desactiva un medicamento sin eliminarlo
// fisicamente, preservando el historial de ventas.
func (s *medicineService) DeactivateMedicine(id uuid.UUID, userID uuid.UUID, ip string) error {
	medicine, err := s.medicineRepo.FindByID(id)
	if err != nil {
		return ErrMedicineNotFound
	}

	if !medicine.IsActive {
		return ErrMedicineAlreadyInactive
	}

	if err := s.medicineRepo.Deactivate(id); err != nil {
		return err
	}

	detail := fmt.Sprintf("medicamento desactivado: %s (ID: %s)", medicine.Name, medicine.ID.String())
	_ = s.auditRepo.Log(userID, "medicine_deactivate", detail, ip)

	return nil
}

// ActivateMedicine reactiva un medicamento previamente desactivado.
func (s *medicineService) ActivateMedicine(id uuid.UUID, userID uuid.UUID, ip string) error {
	medicine, err := s.medicineRepo.FindByID(id)
	if err != nil {
		return ErrMedicineNotFound
	}

	if medicine.IsActive {
		return ErrMedicineAlreadyActive
	}

	if err := s.medicineRepo.Activate(id); err != nil {
		return err
	}

	detail := fmt.Sprintf("medicamento activado: %s (ID: %s)", medicine.Name, medicine.ID.String())
	_ = s.auditRepo.Log(userID, "medicine_activate", detail, ip)

	return nil
}

// SearchMedicines cubre RF-014: busqueda de medicamentos con filtros
// multiples (nombre, barcode, laboratorio, precios, stock).
func (s *medicineService) SearchMedicines(params MedicineSearchParams) ([]domain.Medicine, error) {
	return s.medicineRepo.Search(
		params.Name,
		params.Barcode,
		params.Laboratory,
		params.MinPrice,
		params.MaxPrice,
		params.HasStock,
	)
}
