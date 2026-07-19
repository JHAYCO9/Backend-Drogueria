package dto

// RegisterAdjustmentRequest cubre RF-017: datos necesarios para registrar
// un ajuste manual de inventario.
type RegisterAdjustmentRequest struct {
	MedicineID  string `json:"medicine_id" binding:"required"`
	BatchID     string `json:"batch_id" binding:"required"`
	NewQuantity int    `json:"new_quantity" binding:"gte=0"`
	Reason      string `json:"reason" binding:"required"`
}

// InventoryMovementResponse cubre RF-019: representacion JSON de un
// movimiento de inventario para el historial.
type InventoryMovementResponse struct {
	ID           string `json:"id"`
	MedicineID   string `json:"medicine_id"`
	MedicineName string `json:"medicine_name"`
	BatchID      string `json:"batch_id,omitempty"`
	BatchCode    string `json:"batch_code,omitempty"`
	MovementType string `json:"movement_type"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason,omitempty"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	CreatedAt    string `json:"created_at"`
}

// InventoryGeneralItem cubre RF-028: consolidado de inventario por
// medicamento.
type InventoryGeneralItem struct {
	ID                string `json:"id"`
	Barcode           string `json:"barcode"`
	Name              string `json:"name"`
	Laboratory        string `json:"laboratory"`
	TotalStock        int    `json:"total_stock"`
	MinStock          int    `json:"min_stock"`
	IsActive          bool   `json:"is_active"`
	Status            string `json:"status"`
	NextExpiration    string `json:"next_expiration,omitempty"`
	AvailableBatches  int    `json:"available_batches"`
}

// CriticalInventoryItem cubre RF-029: medicamento que cumple al menos
// una condicion critica (stock bajo, proximo a vencer, vencido).
type CriticalInventoryItem struct {
	ID           string   `json:"id"`
	Barcode      string   `json:"barcode"`
	Name         string   `json:"name"`
	Laboratory   string   `json:"laboratory"`
	TotalStock   int      `json:"total_stock"`
	MinStock     int      `json:"min_stock"`
	IsActive     bool     `json:"is_active"`
	Reasons      []string `json:"reasons"`
}
