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
	FindByBarcode(barcode string) (*domain.Medicine, error) // RF-027, listo para cuando llegue el lector
	FindAll() ([]domain.Medicine, error)
	Update(medicine *domain.Medicine) error
	UpdateAlertDays(id uuid.UUID, days int) error
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

// FindByBarcode es la base de RF-027 (buscar medicamento mediante código de
// barras). El lector físico solo produce el string del código — esta
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
			"min_stock":      medicine.MinStock, // RF-020: se actualiza junto con el resto
		}).Error
}
