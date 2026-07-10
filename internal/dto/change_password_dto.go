package dto

// ChangePasswordRequest recibe las tres contraseñas necesarias para el
// flujo de cambio de contraseña (RF-006). El binding de Gin valida que
// ninguna venga vacía y que la nueva tenga al menos 8 caracteres.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}
