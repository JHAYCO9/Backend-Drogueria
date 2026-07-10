package dto

type RegisterUserRequest struct {
	FullName          string  `json:"full_name" binding:"required"`
	DocumentID        string  `json:"document_id" binding:"required"`
	Email             *string `json:"email"`
	Username          string  `json:"username" binding:"required"`
	TemporaryPassword string  `json:"temporary_password" binding:"required,min=8"`
	RoleID            string  `json:"role_id" binding:"required"`
}

type RegisterUserResponse struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type UpdateUserRequest struct {
	FullName string `json:"full_name" binding:"required"`
	Username string `json:"username" binding:"required"`
	RoleID   string `json:"role_id" binding:"required"`
	IsActive bool   `json:"is_active"`
}
