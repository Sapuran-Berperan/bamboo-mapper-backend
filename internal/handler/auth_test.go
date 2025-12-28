package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/auth"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/config"
	appMiddleware "github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/middleware"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/model"
	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/repository"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var testDB *sql.DB
var testQueries *repository.Queries
var testJWTManager *auth.JWTManager

func TestMain(m *testing.M) {
	// Load .env file from project root
	// Tests run from package directory, so we need to find the project root
	projectRoot := findProjectRoot()
	if projectRoot != "" {
		godotenv.Load(filepath.Join(projectRoot, ".env"))
	}

	// Setup: connect to test database
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		// Skip integration tests if no test DB configured
		os.Exit(0)
	}

	var err error
	testDB, err = sql.Open("postgres", dbURL)
	if err != nil {
		panic("failed to connect to test database: " + err.Error())
	}

	if err := testDB.Ping(); err != nil {
		panic("failed to ping test database: " + err.Error())
	}

	testQueries = repository.New(testDB)

	// Initialize JWT manager for tests
	testCfg := &config.Config{
		JWTSecret:          "test-secret-key-for-testing-min-32-chars",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	testJWTManager = auth.NewJWTManager(testCfg)

	// Run tests
	code := m.Run()

	// Teardown
	testDB.Close()
	os.Exit(code)
}

func cleanupUsers(t *testing.T) {
	_, err := testDB.Exec("DELETE FROM refresh_tokens")
	if err != nil {
		t.Fatalf("failed to cleanup refresh_tokens table: %v", err)
	}
	_, err = testDB.Exec("DELETE FROM users")
	if err != nil {
		t.Fatalf("failed to cleanup users table: %v", err)
	}
}

func TestAuthHandler_Register_Success(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)

	reqBody := `{"email": "test@example.com", "name": "Test User", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Errorf("expected success=true, got false")
	}

	if response.Meta.Message != "User registered successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	// Verify user data in response
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", response.Data)
	}

	if data["email"] != "test@example.com" {
		t.Errorf("expected email 'test@example.com', got %v", data["email"])
	}

	if data["name"] != "Test User" {
		t.Errorf("expected name 'Test User', got %v", data["name"])
	}

	if data["role"] != "user" {
		t.Errorf("expected role 'user', got %v", data["role"])
	}

	// Verify password is not in response
	if _, exists := data["password"]; exists {
		t.Error("password should not be in response")
	}
	if _, exists := data["password_hash"]; exists {
		t.Error("password_hash should not be in response")
	}

	// Verify user exists in database
	user, err := testQueries.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("user not found in database: %v", err)
	}

	if user.Name != "Test User" {
		t.Errorf("expected name in DB 'Test User', got %s", user.Name)
	}
}

func TestAuthHandler_Register_DuplicateEmail(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)

	// First registration
	reqBody := `{"email": "duplicate@example.com", "name": "First User", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("first registration failed: %s", rr.Body.String())
	}

	// Second registration with same email
	reqBody = `{"email": "duplicate@example.com", "name": "Second User", "password": "password456"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr = httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected status %d, got %d: %s", http.StatusConflict, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false for duplicate email")
	}

	if response.Meta.Message != "Email already registered" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Register_ValidationErrors(t *testing.T) {
	handler := NewAuthHandler(testQueries, testJWTManager)

	tests := []struct {
		name           string
		reqBody        string
		expectedFields []string
	}{
		{
			name:           "empty email",
			reqBody:        `{"email": "", "name": "Test", "password": "password123"}`,
			expectedFields: []string{"email"},
		},
		{
			name:           "invalid email",
			reqBody:        `{"email": "invalid", "name": "Test", "password": "password123"}`,
			expectedFields: []string{"email"},
		},
		{
			name:           "empty name",
			reqBody:        `{"email": "test@example.com", "name": "", "password": "password123"}`,
			expectedFields: []string{"name"},
		},
		{
			name:           "short password",
			reqBody:        `{"email": "test@example.com", "name": "Test", "password": "short"}`,
			expectedFields: []string{"password"},
		},
		{
			name:           "multiple errors",
			reqBody:        `{"email": "invalid", "name": "", "password": "short"}`,
			expectedFields: []string{"email", "name", "password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.Register(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
			}

			var response Response
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to parse response: %v", err)
			}

			if response.Meta.Success {
				t.Error("expected success=false")
			}

			if response.Meta.Message != "Validation failed" {
				t.Errorf("unexpected message: %s", response.Meta.Message)
			}

			for _, field := range tt.expectedFields {
				if _, exists := response.Meta.Details[field]; !exists {
					t.Errorf("expected error for field %s, got details: %v", field, response.Meta.Details)
				}
			}
		})
	}
}

func TestAuthHandler_Register_InvalidJSON(t *testing.T) {
	handler := NewAuthHandler(testQueries, testJWTManager)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Invalid request body" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Register_EmailNormalization(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)

	// Register with uppercase email
	reqBody := `{"email": "TEST@EXAMPLE.COM", "name": "Test User", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("registration failed: %s", rr.Body.String())
	}

	// Verify email is lowercased in database
	user, err := testQueries.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("user not found with lowercase email: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf("expected lowercase email, got %s", user.Email)
	}
}

// ResponseWithUserData is a helper for parsing response with user data
type ResponseWithUserData struct {
	Meta Meta               `json:"meta"`
	Data *model.UserResponse `json:"data"`
}

// findProjectRoot walks up the directory tree to find the project root (where .env lives)
func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// createTestUser creates a user for login tests and returns the email and password
func createTestUser(t *testing.T, handler *AuthHandler) {
	reqBody := `{"email": "login@example.com", "name": "Login User", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to create test user: %s", rr.Body.String())
	}
}

func TestAuthHandler_Login_Success(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	reqBody := `{"email": "login@example.com", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Errorf("expected success=true, got false")
	}

	if response.Meta.Message != "Login successful" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	// Verify login response data
	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", response.Data)
	}

	if data["access_token"] == nil || data["access_token"] == "" {
		t.Error("expected access_token in response")
	}

	if data["refresh_token"] == nil || data["refresh_token"] == "" {
		t.Error("expected refresh_token in response")
	}

	if data["token_type"] != "Bearer" {
		t.Errorf("expected token_type 'Bearer', got %v", data["token_type"])
	}

	if data["expires_in"] == nil {
		t.Error("expected expires_in in response")
	}

	// Verify user data
	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected user to be a map, got %T", data["user"])
	}

	if user["email"] != "login@example.com" {
		t.Errorf("expected email 'login@example.com', got %v", user["email"])
	}
}

func TestAuthHandler_Login_InvalidEmail(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	reqBody := `{"email": "nonexistent@example.com", "password": "password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	// Generic message to prevent email enumeration
	if response.Meta.Message != "Invalid email or password" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Login_InvalidPassword(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	reqBody := `{"email": "login@example.com", "password": "wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Login(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	// Same message as invalid email
	if response.Meta.Message != "Invalid email or password" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Refresh_Success(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	// First, login to get tokens
	loginBody := `{"email": "login@example.com", "password": "password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("login failed: %s", loginRR.Body.String())
	}

	var loginResponse Response
	if err := json.Unmarshal(loginRR.Body.Bytes(), &loginResponse); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}

	loginData := loginResponse.Data.(map[string]interface{})
	refreshToken := loginData["refresh_token"].(string)

	// Now use refresh token to get new tokens
	refreshBody := `{"refresh_token": "` + refreshToken + `"}`
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshRR := httptest.NewRecorder()
	handler.Refresh(refreshRR, refreshReq)

	if refreshRR.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, refreshRR.Code, refreshRR.Body.String())
	}

	var refreshResponse Response
	if err := json.Unmarshal(refreshRR.Body.Bytes(), &refreshResponse); err != nil {
		t.Fatalf("failed to parse refresh response: %v", err)
	}

	if !refreshResponse.Meta.Success {
		t.Error("expected success=true")
	}

	if refreshResponse.Meta.Message != "Token refreshed successfully" {
		t.Errorf("unexpected message: %s", refreshResponse.Meta.Message)
	}

	refreshData := refreshResponse.Data.(map[string]interface{})

	if refreshData["access_token"] == nil || refreshData["access_token"] == "" {
		t.Error("expected new access_token")
	}

	if refreshData["refresh_token"] == nil || refreshData["refresh_token"] == "" {
		t.Error("expected new refresh_token")
	}

	// New refresh token should be different (rotation)
	if refreshData["refresh_token"] == refreshToken {
		t.Error("expected new refresh token to be different from old one")
	}
}

func TestAuthHandler_Refresh_InvalidToken(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)

	reqBody := `{"refresh_token": "invalid-token"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.Refresh(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Invalid or expired refresh token" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Refresh_TokenRotation(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	// Login to get tokens
	loginBody := `{"email": "login@example.com", "password": "password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)

	var loginResponse Response
	json.Unmarshal(loginRR.Body.Bytes(), &loginResponse)
	loginData := loginResponse.Data.(map[string]interface{})
	refreshToken := loginData["refresh_token"].(string)

	// Use refresh token once
	refreshBody := `{"refresh_token": "` + refreshToken + `"}`
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshRR := httptest.NewRecorder()
	handler.Refresh(refreshRR, refreshReq)

	if refreshRR.Code != http.StatusOK {
		t.Fatalf("first refresh failed: %s", refreshRR.Body.String())
	}

	// Try to use the same refresh token again (should fail - it's revoked)
	refreshReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody))
	refreshReq2.Header.Set("Content-Type", "application/json")

	refreshRR2 := httptest.NewRecorder()
	handler.Refresh(refreshRR2, refreshReq2)

	if refreshRR2.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d for reused token, got %d", http.StatusUnauthorized, refreshRR2.Code)
	}
}

// createTestRouter creates a chi router with the GetMe endpoint protected by JWT middleware
func createTestRouter(handler *AuthHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.JWTAuth(testJWTManager))
			r.Get("/me", handler.GetMe)
		})
	})
	return r
}

// loginAndGetToken logs in and returns the access token
func loginAndGetToken(t *testing.T, handler *AuthHandler) string {
	loginBody := `{"email": "login@example.com", "password": "password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("login failed: %s", loginRR.Body.String())
	}

	var loginResponse Response
	json.Unmarshal(loginRR.Body.Bytes(), &loginResponse)
	loginData := loginResponse.Data.(map[string]interface{})
	return loginData["access_token"].(string)
}

func TestAuthHandler_GetMe_Success(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)
	accessToken := loginAndGetToken(t, handler)

	router := createTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Error("expected success=true")
	}

	if response.Meta.Message != "User retrieved successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", response.Data)
	}

	if data["email"] != "login@example.com" {
		t.Errorf("expected email 'login@example.com', got %v", data["email"])
	}

	if data["name"] != "Login User" {
		t.Errorf("expected name 'Login User', got %v", data["name"])
	}
}

func TestAuthHandler_GetMe_NoToken(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	router := createTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	// No Authorization header

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Authorization header required" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_GetMe_InvalidToken(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	router := createTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Invalid token" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_GetMe_ExpiredToken(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	// Create an expired JWT manager
	expiredCfg := &config.Config{
		JWTSecret:          "test-secret-key-for-testing-min-32-chars",
		AccessTokenExpiry:  -1 * time.Hour, // Already expired
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
	expiredJWTManager := auth.NewJWTManager(expiredCfg)

	// Get user to generate expired token
	user, _ := testQueries.GetUserByEmail(context.Background(), "login@example.com")
	expiredToken, _ := expiredJWTManager.GenerateAccessToken(user.ID, user.Email, user.Role.String)

	router := createTestRouter(handler)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Token has expired" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

// createTestRouterWithLogout creates a chi router with GetMe and Logout endpoints
func createTestRouterWithLogout(handler *AuthHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appMiddleware.JWTAuth(testJWTManager))
			r.Get("/me", handler.GetMe)
			r.Post("/logout", handler.Logout)
		})
	})
	return r
}

func TestAuthHandler_Logout_Success(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	// Login to get tokens
	loginBody := `{"email": "login@example.com", "password": "password123"}`
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")

	loginRR := httptest.NewRecorder()
	handler.Login(loginRR, loginReq)

	var loginResponse Response
	json.Unmarshal(loginRR.Body.Bytes(), &loginResponse)
	loginData := loginResponse.Data.(map[string]interface{})
	accessToken := loginData["access_token"].(string)
	refreshToken := loginData["refresh_token"].(string)

	// Call logout endpoint
	router := createTestRouterWithLogout(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Error("expected success=true")
	}

	if response.Meta.Message != "Logged out successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	// Verify refresh token is now invalid
	refreshBody := `{"refresh_token": "` + refreshToken + `"}`
	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody))
	refreshReq.Header.Set("Content-Type", "application/json")

	refreshRR := httptest.NewRecorder()
	handler.Refresh(refreshRR, refreshReq)

	if refreshRR.Code != http.StatusUnauthorized {
		t.Errorf("expected refresh to fail after logout, got status %d", refreshRR.Code)
	}
}

func TestAuthHandler_Logout_NoToken(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	router := createTestRouterWithLogout(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	// No Authorization header

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Authorization header required" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestAuthHandler_Logout_RevokesAllSessions(t *testing.T) {
	cleanupUsers(t)

	handler := NewAuthHandler(testQueries, testJWTManager)
	createTestUser(t, handler)

	// Login twice to create two sessions
	loginBody := `{"email": "login@example.com", "password": "password123"}`

	// First login
	loginReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq1.Header.Set("Content-Type", "application/json")
	loginRR1 := httptest.NewRecorder()
	handler.Login(loginRR1, loginReq1)

	var loginResponse1 Response
	json.Unmarshal(loginRR1.Body.Bytes(), &loginResponse1)
	loginData1 := loginResponse1.Data.(map[string]interface{})
	accessToken := loginData1["access_token"].(string)
	refreshToken1 := loginData1["refresh_token"].(string)

	// Second login
	loginReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(loginBody))
	loginReq2.Header.Set("Content-Type", "application/json")
	loginRR2 := httptest.NewRecorder()
	handler.Login(loginRR2, loginReq2)

	var loginResponse2 Response
	json.Unmarshal(loginRR2.Body.Bytes(), &loginResponse2)
	loginData2 := loginResponse2.Data.(map[string]interface{})
	refreshToken2 := loginData2["refresh_token"].(string)

	// Logout using first session's access token
	router := createTestRouterWithLogout(handler)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("logout failed: %s", rr.Body.String())
	}

	// Verify BOTH refresh tokens are now invalid
	refreshBody1 := `{"refresh_token": "` + refreshToken1 + `"}`
	refreshReq1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody1))
	refreshReq1.Header.Set("Content-Type", "application/json")
	refreshRR1 := httptest.NewRecorder()
	handler.Refresh(refreshRR1, refreshReq1)

	if refreshRR1.Code != http.StatusUnauthorized {
		t.Errorf("expected first refresh token to be revoked, got status %d", refreshRR1.Code)
	}

	refreshBody2 := `{"refresh_token": "` + refreshToken2 + `"}`
	refreshReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody2))
	refreshReq2.Header.Set("Content-Type", "application/json")
	refreshRR2 := httptest.NewRecorder()
	handler.Refresh(refreshRR2, refreshReq2)

	if refreshRR2.Code != http.StatusUnauthorized {
		t.Errorf("expected second refresh token to be revoked, got status %d", refreshRR2.Code)
	}
}
