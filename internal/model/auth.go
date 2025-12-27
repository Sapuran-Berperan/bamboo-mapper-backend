package model

import "strings"

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks if the login request is valid
func (r *LoginRequest) Validate() map[string]string {
	errors := make(map[string]string)

	r.Email = strings.TrimSpace(r.Email)

	if r.Email == "" {
		errors["email"] = "email is required"
	}

	if r.Password == "" {
		errors["password"] = "password is required"
	}

	return errors
}

// LoginResponse represents the login response
type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
}

// RefreshRequest represents the refresh token request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Validate checks if the refresh request is valid
func (r *RefreshRequest) Validate() map[string]string {
	errors := make(map[string]string)

	if strings.TrimSpace(r.RefreshToken) == "" {
		errors["refresh_token"] = "refresh_token is required"
	}

	return errors
}

// RefreshResponse represents the refresh token response
type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}
