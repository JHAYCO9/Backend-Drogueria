package service

import (
	"github.com/drogueria/drogueria-micky-backend/internal/dto"
	"github.com/drogueria/drogueria-micky-backend/internal/repository"
)

// AuditService expone la consulta de registros de auditoría (RF-010).
type AuditService interface {
	GetAllAuditLogs() ([]dto.AuditLogResponse, error)
}

type auditService struct {
	auditRepo repository.AuditRepository
	userRepo  repository.UserRepository
}

func NewAuditService(auditRepo repository.AuditRepository, userRepo repository.UserRepository) AuditService {
	return &auditService{auditRepo: auditRepo, userRepo: userRepo}
}

// GetAllAuditLogs recupera todos los registros de auditoría y enriquece
// cada entrada con el username del usuario que realizó la acción, para
// que el frontend no tenga que resolver UUIDs.
func (s *auditService) GetAllAuditLogs() ([]dto.AuditLogResponse, error) {
	logs, err := s.auditRepo.FindAll()
	if err != nil {
		return nil, err
	}

	// Cachear usernames para evitar consultas repetidas a la BD
	usernameCache := make(map[string]string)

	result := make([]dto.AuditLogResponse, 0, len(logs))
	for _, entry := range logs {
		userIDStr := entry.UserID.String()

		// Buscar el username en caché o en la BD
		username, ok := usernameCache[userIDStr]
		if !ok {
			user, err := s.userRepo.FindByID(entry.UserID)
			if err != nil {
				username = "usuario eliminado"
			} else {
				username = user.Username
			}
			usernameCache[userIDStr] = username
		}

		result = append(result, dto.AuditLogResponse{
			ID:        entry.ID.String(),
			UserID:    userIDStr,
			Username:  username,
			Action:    entry.Action,
			Detail:    entry.Detail,
			IPAddress: entry.IPAddress,
			CreatedAt: entry.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return result, nil
}
