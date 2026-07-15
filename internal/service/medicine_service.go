package service

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

var ErrBarcodeTaken = errors.New("ya existe un medicamento registrado con ese código de barras")
var ErrMedicineNotFound = errors.New("medicamento no encontrado")

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

type CriticalMedicine struct {
	Medicine     domain.Medicine
	CurrentStock int
}

type MedicineService interface {
	RegisterMedicine(input RegisterMedicineInput) (*domain.Medicine, error)
	FindByBarcode(barcode string) (*domain.Medicine, error) // RF-027
	ListMedicines() ([]domain.Medicine, error)
	UpdateMinStock(id uuid.UUID, minStock int) error    // RF-020 dedicado
	ListCriticalMedicines() ([]CriticalMedicine, error) // RF-021
	UpdateAlertDays(id uuid.UUID, days int) error       // RF-022
	FindByID(id uuid.UUID) (*domain.Medicine, error)
}

type medicineService struct {
	medicineRepo repository.MedicineRepository
	batchRepo    repository.BatchRepository
}

func (s *medicineService) FindByID(id uuid.UUID) (*domain.Medicine, error) {
	return s.medicineRepo.FindByID(id)
}

func NewMedicineService(medicineRepo repository.MedicineRepository, batchRepo repository.BatchRepository) MedicineService {
	return &medicineService{medicineRepo: medicineRepo, batchRepo: batchRepo}
}

// RegisterMedicine cubre RF-011 completo: valida campos obligatorios
// (a nivel de DTO con `binding`), impide códigos de barras duplicados,
// y registra fecha de creación (automática vía GORM). Incluye MinStock
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

// FindByBarcode es RF-027: recibe el código que capturó el lector físico
// y devuelve el medicamento correspondiente. No se puede probar de punta
// a punta sin el lector, pero la lógica queda lista — el endpoint acepta
// el código como string plano, que es exactamente lo que cualquier lector
// USB tipo "teclado" entrega al sistema.
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

// UpdateMinStock cubre RF-020 como endpoint independiente. La validación
// de "mayor o igual a cero" ya la impone el DTO (`binding:"gte=0"`), pero
// se repite aquí como resguardo por si algún día se llama sin pasar por
// el binding de Gin.
func (s *medicineService) UpdateMinStock(id uuid.UUID, minStock int) error {
	if minStock < 0 {
		return errors.New("el stock mínimo debe ser mayor o igual a cero")
	}

	medicine, err := s.medicineRepo.FindByID(id)
	if err != nil {
		return ErrMedicineNotFound
	}

	medicine.MinStock = minStock
	return s.medicineRepo.Update(medicine)
}

// ListCriticalMedicines cubre RF-021: medicamentos cuyo stock actual
// (suma de todos sus lotes con existencias) es menor o igual a su stock
// mínimo configurado (RF-020). El envío por WhatsApp (Épica 7) consumirá
// esta misma lista más adelante — aquí solo se deja la data lista.
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
