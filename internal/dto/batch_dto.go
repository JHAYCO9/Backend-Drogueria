package dto

// RegisterBatchRequest cubre RF-015: registro inicial de un lote.
type RegisterBatchRequest struct {
	MedicineID     string `json:"medicine_id" binding:"required"`
	Quantity       int    `json:"quantity" binding:"gte=0"`
	ReceivedAt     string `json:"received_at" binding:"required"`     // formato "2006-01-02"
	ExpirationDate string `json:"expiration_date" binding:"required"` // formato "2006-01-02"
}

type RegisterIngresoRequest struct {
	BatchID  string `json:"batch_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,gt=0"`
	SaleUnit string `json:"sale_unit" binding:"required,oneof=pastilla sello caja"`
}

type BatchResponse struct {
	ID             string `json:"id"`
	MedicineID     string `json:"medicine_id"`
	MedicineName   string `json:"medicine_name"`
	BatchCode      string `json:"batch_code"`
	Quantity       int    `json:"quantity"`
	ReceivedAt     string `json:"received_at"`
	ExpirationDate string `json:"expiration_date"`
	Status         string `json:"status"` // "vigente", "proximo_a_vencer", "vencido" — RF-018
	DaysRemaining  int    `json:"days_remaining"`
}

type UpdateAlertDaysRequest struct {
	Days int `json:"days" binding:"required,gt=0"` // RF-022
}
