package dto

// AuditLogResponse es la representación JSON de un registro de auditoría
// que se devuelve al frontend cuando el Administrador consulta el
// historial de acciones (RF-010).
type AuditLogResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Action    string `json:"action"`
	Detail    string `json:"detail"`
	IPAddress string `json:"ip_address"`
	CreatedAt string `json:"created_at"`
}
