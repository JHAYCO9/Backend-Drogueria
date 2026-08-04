package dto

// RF-092: Motivos válidos de devolución.
// El cliente debe enviar exactamente uno de estos valores en el campo "reason".
var ValidReturnReasons = []string{
	"Error en la venta",
	"Error en el producto entregado",
	"Producto defectuoso",
	"Otro",
}

// IsValidReturnReason valida que el motivo enviado sea uno de los permitidos (RF-092).
func IsValidReturnReason(reason string) bool {
	for _, r := range ValidReturnReasons {
		if r == reason {
			return true
		}
	}
	return false
}

// RegisterReturnRequest cubre RF-089 + RF-090 + RF-092:
// - sale_item_id: ítem específico de la venta
// - quantity: unidades a devolver (RF-090: se valida que no supere lo vendido)
// - reason: obligatorio y debe ser uno de los motivos predefinidos (RF-092)
type RegisterReturnRequest struct {
	SaleItemID string `json:"sale_item_id" binding:"required"`
	Quantity   int    `json:"quantity" binding:"required,gt=0"`
	Reason     string `json:"reason" binding:"required"`
}

type VoidReturnRequest struct {
	Reason string `json:"reason" binding:"required,min=3"`
}

type ReturnResponse struct {
	ID           string `json:"id"`
	SaleID       string `json:"sale_id"`
	SaleNumber   int    `json:"sale_number"`
	MedicineName string `json:"medicine_name"`
	Quantity     int    `json:"quantity"`
	Reason       string `json:"reason"`
	Username     string `json:"username"`
	IsVoided     bool   `json:"is_voided"`
	VoidedReason string `json:"voided_reason,omitempty"`
	VoidedAt     string `json:"voided_at,omitempty"`
	VoidedByName string `json:"voided_by_name,omitempty"`
	CreatedAt    string `json:"created_at"`
}
