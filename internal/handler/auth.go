package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	queries *repository.Queries
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(queries *repository.Queries) *AuthHandler {
	return &AuthHandler{queries: queries}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req model.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Validate input
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		respondError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process registration", nil)
		return
	}

	// Create user in database
	user, err := h.queries.CreateUser(r.Context(), repository.CreateUserParams{
		Email:        strings.ToLower(req.Email),
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Role:         sql.NullString{String: "user", Valid: true},
	})
	if err != nil {
		// Check for duplicate email (PostgreSQL unique violation)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			respondError(w, http.StatusConflict, "Email already registered", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	// Return created user (without password)
	response := model.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role.String,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}

	respondSuccess(w, http.StatusCreated, "User registered successfully", response)
}
