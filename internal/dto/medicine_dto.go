package dto

// RegisterMedicineRequest cubre RF-011 (registro) y RF-020 (stock mínimo
// como parte del mismo formulario, ya que es un campo del medicamento).
type RegisterMedicineRequest struct {
	Barcode       string  `json:"barcode" binding:"required"`
	Name          string  `json:"name" binding:"required"`
	Laboratory    string  `json:"laboratory"`
	Concentration string  `json:"concentration"`
	Presentation  string  `json:"presentation"`
	PricePerPill  float64 `json:"price_per_pill" binding:"gte=0"`
	PricePerSeal  float64 `json:"price_per_seal" binding:"gte=0"`
	PricePerBox   float64 `json:"price_per_box" binding:"gte=0"`
	MinStock      int     `json:"min_stock" binding:"gte=0"` // RF-020: debe ser >= 0
}

type MedicineResponse struct {
	ID            string  `json:"id"`
	Barcode       string  `json:"barcode"`
	Name          string  `json:"name"`
	Laboratory    string  `json:"laboratory"`
	Concentration string  `json:"concentration"`
	Presentation  string  `json:"presentation"`
	PricePerPill  float64 `json:"price_per_pill"`
	PricePerSeal  float64 `json:"price_per_seal"`
	PricePerBox   float64 `json:"price_per_box"`
	MinStock      int     `json:"min_stock"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
}

// UpdateMinStockRequest — endpoint dedicado para RF-020, por si en el
// futuro quieren ajustar SOLO el stock mínimo sin pasar por el formulario
// completo de edición del medicamento.
type UpdateMinStockRequest struct {
	MinStock int `json:"min_stock" binding:"gte=0"`
}
