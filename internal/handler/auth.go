package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/auth"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/middleware"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	queries    *repository.Queries
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(queries *repository.Queries, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		queries:    queries,
		jwtManager: jwtManager,
	}
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

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Validate input
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		respondError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	// Fetch user by email
	user, err := h.queries.GetUserByEmail(r.Context(), strings.ToLower(req.Email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusUnauthorized, "Invalid email or password", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to process login", nil)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid email or password", nil)
		return
	}

	// Generate access token
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Role.String)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	// Generate and store refresh token
	rawRefreshToken, tokenHash, expiresAt, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	_, err = h.queries.CreateRefreshToken(r.Context(), repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		UserAgent: sql.NullString{String: r.UserAgent(), Valid: r.UserAgent() != ""},
		IpAddress: sql.NullString{String: getClientIP(r), Valid: true},
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session", nil)
		return
	}

	// Return response
	response := model.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.jwtManager.GetAccessTokenExpiry().Seconds()),
		User: model.UserResponse{
			ID:        user.ID,
			Email:     user.Email,
			Name:      user.Name,
			Role:      user.Role.String,
			CreatedAt: user.CreatedAt.Time,
			UpdatedAt: user.UpdatedAt.Time,
		},
	}

	respondSuccess(w, http.StatusOK, "Login successful", response)
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	// Parse request body
	var req model.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body", nil)
		return
	}

	// Validate input
	if validationErrors := req.Validate(); len(validationErrors) > 0 {
		respondError(w, http.StatusBadRequest, "Validation failed", validationErrors)
		return
	}

	// Hash the provided token and look it up
	tokenHash := auth.HashRefreshToken(req.RefreshToken)

	storedToken, err := h.queries.GetRefreshTokenByHash(r.Context(), tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to validate token", nil)
		return
	}

	// Revoke the old refresh token (rotation)
	if err := h.queries.RevokeRefreshToken(r.Context(), tokenHash); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to process refresh", nil)
		return
	}

	// Fetch user data for new access token
	user, err := h.queries.GetUserByID(r.Context(), storedToken.UserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch user", nil)
		return
	}

	// Generate new access token
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Role.String)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	// Generate new refresh token
	rawRefreshToken, newTokenHash, expiresAt, err := h.jwtManager.GenerateRefreshToken()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate token", nil)
		return
	}

	_, err = h.queries.CreateRefreshToken(r.Context(), repository.CreateRefreshTokenParams{
		UserID:    storedToken.UserID,
		TokenHash: newTokenHash,
		ExpiresAt: expiresAt,
		UserAgent: sql.NullString{String: r.UserAgent(), Valid: r.UserAgent() != ""},
		IpAddress: sql.NullString{String: getClientIP(r), Valid: true},
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create session", nil)
		return
	}

	// Return response
	response := model.RefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(h.jwtManager.GetAccessTokenExpiry().Seconds()),
	}

	respondSuccess(w, http.StatusOK, "Token refreshed successfully", response)
}

// GetMe returns the currently authenticated user
func (h *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	// Get claims from context (set by JWT middleware)
	claims, ok := middleware.GetClaims(r.Context())
	if !ok {
		respondError(w, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}

	// Fetch fresh user data from database
	user, err := h.queries.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusUnauthorized, "User not found", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "Failed to fetch user", nil)
		return
	}

	response := model.UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role.String,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}

	respondSuccess(w, http.StatusOK, "User retrieved successfully", response)
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for reverse proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}
	return ip
}
