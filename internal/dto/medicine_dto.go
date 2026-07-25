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
	MinStock      int     `json:"min_stock" binding:"gte=0"`
	HasIva        bool    `json:"has_iva"`
	SellsByPill   bool    `json:"sells_by_pill"`
	SellsBySeal   bool    `json:"sells_by_seal"`
	SellsByBox    bool    `json:"sells_by_box"`
	UnitsPerSeal  int     `json:"units_per_seal" binding:"gte=1"`
	UnitsPerBox   int     `json:"units_per_box" binding:"gte=1"`
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
	HasIva        bool    `json:"has_iva"`
	SellsByPill   bool    `json:"sells_by_pill"`
	SellsBySeal   bool    `json:"sells_by_seal"`
	SellsByBox    bool    `json:"sells_by_box"`
	UnitsPerSeal  int     `json:"units_per_seal"`
	UnitsPerBox   int     `json:"units_per_box"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
}

// UpdateMinStockRequest -- endpoint dedicado para RF-020, por si en el
// futuro quieren ajustar SOLO el stock minimo sin pasar por el formulario
// completo de edicion del medicamento.
type UpdateMinStockRequest struct {
	MinStock int `json:"min_stock" binding:"gte=0"`
}

// UpdateMedicineRequest cubre RF-012: campos que el administrador puede
// modificar de un medicamento existente.
type UpdateMedicineRequest struct {
	Name          *string  `json:"name"`
	Presentation  *string  `json:"presentation"`
	Concentration *string  `json:"concentration"`
	PricePerPill  *float64 `json:"price_per_pill" binding:"omitempty,gte=0"`
	PricePerSeal  *float64 `json:"price_per_seal" binding:"omitempty,gte=0"`
	PricePerBox   *float64 `json:"price_per_box" binding:"omitempty,gte=0"`
	MinStock      *int     `json:"min_stock" binding:"omitempty,gte=0"`
	HasIva        bool     `json:"has_iva"`
	SellsByPill   bool     `json:"sells_by_pill"`
	SellsBySeal   bool     `json:"sells_by_seal"`
	SellsByBox    bool     `json:"sells_by_box"`
	UnitsPerSeal  int      `json:"units_per_seal" binding:"gte=1"`
	UnitsPerBox   int      `json:"units_per_box" binding:"gte=1"`
}

// MedicineSearchResponse cubre RF-014: respuesta enriquecida con
// existencias totales para la busqueda de medicamentos.
type MedicineSearchResponse struct {
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
	TotalStock    int     `json:"total_stock"`
	IsActive      bool    `json:"is_active"`
	CreatedAt     string  `json:"created_at"`
}
