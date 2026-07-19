package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/drogueria/drogueria-micky-backend/internal/domain"
)

type MedicineRepository interface {
	Create(medicine *domain.Medicine) error
	ExistsByBarcode(barcode string) (bool, error)
	FindByID(id uuid.UUID) (*domain.Medicine, error)
	FindByBarcode(barcode string) (*domain.Medicine, error)
	FindAll() ([]domain.Medicine, error)
	FindAllIncludingInactive() ([]domain.Medicine, error)
	Update(medicine *domain.Medicine) error
	UpdateAlertDays(id uuid.UUID, days int) error
	Deactivate(id uuid.UUID) error
	Activate(id uuid.UUID) error
	Search(name, barcode, laboratory string, minPrice, maxPrice *float64, hasStock *bool) ([]domain.Medicine, error)
}

type medicineRepository struct {
	db *gorm.DB
}

func NewMedicineRepository(db *gorm.DB) MedicineRepository {
	return &medicineRepository{db: db}
}

func (r *medicineRepository) UpdateAlertDays(id uuid.UUID, days int) error {
	return r.db.Model(&domain.Medicine{}).Where("id = ?", id).
		UpdateColumn("expiration_alert_days", days).Error
}

func (r *medicineRepository) Create(medicine *domain.Medicine) error {
	return r.db.Create(medicine).Error
}

func (r *medicineRepository) ExistsByBarcode(barcode string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Medicine{}).Where("barcode = ?", barcode).Count(&count).Error
	return count > 0, err
}

func (r *medicineRepository) FindByID(id uuid.UUID) (*domain.Medicine, error) {
	var medicine domain.Medicine
	if err := r.db.First(&medicine, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &medicine, nil
}

// FindByBarcode es la base de RF-027 (buscar medicamento mediante codigo de
// barras). El lector fisico solo produce el string del codigo -- esta
// consulta es exactamente lo que el endpoint de escaneo va a llamar.
func (r *medicineRepository) FindByBarcode(barcode string) (*domain.Medicine, error) {
	var medicine domain.Medicine
	if err := r.db.First(&medicine, "barcode = ? AND is_active = ?", barcode, true).Error; err != nil {
		return nil, err
	}
	return &medicine, nil
}

func (r *medicineRepository) FindAll() ([]domain.Medicine, error) {
	var medicines []domain.Medicine
	err := r.db.Where("is_active = ?", true).Order("name").Find(&medicines).Error
	return medicines, err
}

// FindAllIncludingInactive retorna todos los medicamentos sin filtrar por
// estado. Se usa en RF-028 (inventario general) donde el administrador
// necesita ver tambien los inactivos.
func (r *medicineRepository) FindAllIncludingInactive() ([]domain.Medicine, error) {
	var medicines []domain.Medicine
	err := r.db.Order("name").Find(&medicines).Error
	return medicines, err
}

func (r *medicineRepository) Update(medicine *domain.Medicine) error {
	return r.db.Model(&domain.Medicine{}).Where("id = ?", medicine.ID).
		Updates(map[string]interface{}{
			"name":           medicine.Name,
			"laboratory":     medicine.Laboratory,
			"concentration":  medicine.Concentration,
			"presentation":   medicine.Presentation,
			"price_per_pill": medicine.PricePerPill,
			"price_per_seal": medicine.PricePerSeal,
			"price_per_box":  medicine.PricePerBox,
			"min_stock":      medicine.MinStock,
		}).Error
}

// Deactivate cubre RF-013: soft delete cambiando is_active a false.
// Nunca se elimina fisicamente el registro.
func (r *medicineRepository) Deactivate(id uuid.UUID) error {
	return r.db.Model(&domain.Medicine{}).Where("id = ?", id).
		UpdateColumn("is_active", false).Error
}

// Activate reactiva un medicamento previamente desactivado.
func (r *medicineRepository) Activate(id uuid.UUID) error {
	return r.db.Model(&domain.Medicine{}).Where("id = ?", id).
		UpdateColumn("is_active", true).Error
}

// Search cubre RF-014: busqueda avanzada con filtros multiples.
func (r *medicineRepository) Search(name, barcode, laboratory string, minPrice, maxPrice *float64, hasStock *bool) ([]domain.Medicine, error) {
	var medicines []domain.Medicine
	query := r.db.Model(&domain.Medicine{})

	if name != "" {
		query = query.Where("LOWER(name) LIKE LOWER(?)", "%"+name+"%")
	}
	if barcode != "" {
		query = query.Where("barcode = ?", barcode)
	}
	if laboratory != "" {
		query = query.Where("LOWER(laboratory) LIKE LOWER(?)", "%"+laboratory+"%")
	}
	if minPrice != nil {
		query = query.Where("price_per_pill >= ?", *minPrice)
	}
	if maxPrice != nil {
		query = query.Where("price_per_pill <= ?", *maxPrice)
	}

	err := query.Order("name").Find(&medicines).Error
	return medicines, err
}
