package dto

import "encoding/json"

type SyncEventRequest struct {
	ID              string          `json:"id" binding:"required"`
	EntityType      string          `json:"entity_type" binding:"required"`
	EntityID        string          `json:"entity_id" binding:"required"`
	Operation       string          `json:"operation" binding:"required"`
	Payload         json.RawMessage `json:"payload" binding:"required"`
	ClientCreatedAt string          `json:"client_created_at" binding:"required"`
}
