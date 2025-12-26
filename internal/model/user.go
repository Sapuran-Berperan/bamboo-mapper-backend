package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// Validate checks if the request is valid and returns field errors
func (r *RegisterRequest) Validate() map[string]string {
	errors := make(map[string]string)

	r.Email = strings.TrimSpace(r.Email)
	r.Name = strings.TrimSpace(r.Name)

	if r.Email == "" {
		errors["email"] = "email is required"
	} else if !isValidEmail(r.Email) {
		errors["email"] = "invalid email format"
	}

	if r.Name == "" {
		errors["name"] = "name is required"
	} else if len(r.Name) > 100 {
		errors["name"] = "name must be 100 characters or less"
	}

	if r.Password == "" {
		errors["password"] = "password is required"
	} else if len(r.Password) < 8 {
		errors["password"] = "password must be at least 8 characters"
	}

	return errors
}

// UserResponse is the API response for user data (excludes password)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 255 {
		return false
	}
	atIndex := strings.Index(email, "@")
	if atIndex < 1 {
		return false
	}
	dotIndex := strings.LastIndex(email, ".")
	return dotIndex > atIndex+1 && dotIndex < len(email)-1
}
