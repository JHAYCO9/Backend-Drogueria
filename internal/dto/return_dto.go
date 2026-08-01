package dto

// RegisterReturnRequest cubre RF-089: selecciona la venta (vía saleId
// en la URL), el ítem específico de esa venta, la cantidad a devolver
// (en unidades sueltas) y el motivo.
type RegisterReturnRequest struct {
	SaleItemID string `json:"sale_item_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
	Reason     string `json:"reason" binding:"required"`
}

type ReturnResponse struct {
	ID           string `json:"id"`
	SaleID       string `json:"sale_id"`
	SaleNumber   int    `json:"sale_number"`
	MedicineName string `json:"medicine_name"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
	Username     string `json:"username"`
	CreatedAt    string `json:"created_at"`
}
