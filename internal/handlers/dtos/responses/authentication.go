package responses

import "time"

type AuthUser struct {
	ID              string     `json:"id" example:"00000000-0000-4000-8000-000000000001"`
	Email           string     `json:"email" example:"user@example.com"`
	DisplayName     string     `json:"display_name" example:"Example User"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

type LoginResponse struct {
	AccessToken string    `json:"access_token" example:"<opaque-access-token>"`
	TokenType   string    `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time `json:"expires_at"`
	User        AuthUser  `json:"user"`
}
