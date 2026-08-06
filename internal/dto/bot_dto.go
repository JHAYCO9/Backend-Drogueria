package dto

type BotQueryRequest struct {
	Command string `json:"command" binding:"required"`
}

type BotQueryResponse struct {
	Command string `json:"command"`
	Result  string `json:"result"`
}

type BotAuthorizedUserResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	CreatedAt string `json:"created_at"`
}

type BotQueryLogResponse struct {
	Username  string `json:"username"`
	Command   string `json:"command"`
	Result    string `json:"result"`
	CreatedAt string `json:"created_at"`
}

type BotConfigResponse struct {
	IsEnabled      bool   `json:"is_enabled"`
	WelcomeMessage string `json:"welcome_message"`
}

type UpdateBotConfigRequest struct {
	IsEnabled      bool   `json:"is_enabled"`
	WelcomeMessage string `json:"welcome_message"`
}

type GrantBotAccessRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
