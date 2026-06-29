package api

// Swagger DTOs (documentation only; handlers may use internal types).

type swaggerTokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"3600"`
}

type swaggerGuestTokenResponse struct {
	GuestToken string `json:"guest_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ExpiresIn  int64  `json:"expires_in" example:"86400"`
}

type swaggerRegisterRequest struct {
	Email    string  `json:"email" example:"user@example.com"`
	Password string  `json:"password" example:"secret123"`
	Username *string `json:"username,omitempty" example:"Player"`
}

type swaggerLoginRequest struct {
	Email    string `json:"email" example:"user@example.com"`
	Password string `json:"password" example:"secret123"`
}

type swaggerRefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type swaggerGuestRequest struct {
	DeviceID string `json:"device_id" example:"00000000-0000-0000-0000-000000000001"`
}

type swaggerUserProfile struct {
	ID        string  `json:"id" example:"00000000-0000-0000-0000-000000000001"`
	Email     string  `json:"email" example:"user@example.com"`
	Username  *string `json:"username,omitempty"`
	Tier      string  `json:"tier" example:"free"`
	CreatedAt string  `json:"created_at" example:"2026-06-10T12:00:00Z"`
}

type swaggerHealthStatus struct {
	Status string `json:"status" example:"ok"`
}

type swaggerErrorResponse struct {
	Error swaggerErrorBody `json:"error"`
}

type swaggerErrorBody struct {
	Code    string `json:"code" example:"VALIDATION_ERROR"`
	Message string `json:"message" example:"invalid request"`
}
