package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/auth"
	appMiddleware "github.com/Sapuran-Berperan/bamboo-mapper-backend/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Note: This test file uses the shared testDB, testQueries, and testJWTManager from auth_test.go
// which are initialized in TestMain

func cleanupMarkers(t *testing.T) {
	_, err := testDB.Exec("DELETE FROM markers")
	if err != nil {
		t.Fatalf("failed to cleanup markers table: %v", err)
	}
}

// createTestMarker creates a marker for testing
func createTestMarker(t *testing.T, creatorID uuid.UUID) uuid.UUID {
	var markerID uuid.UUID
	err := testDB.QueryRow(`
		INSERT INTO markers (creator_id, short_code, name, latitude, longitude, description, strain, quantity)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, creatorID, "TEST001", "Test Bamboo", "-7.12345678", "110.12345678", "Test description", "Bambusa vulgaris", 50).Scan(&markerID)
	if err != nil {
		t.Fatalf("failed to create test marker: %v", err)
	}
	return markerID
}

// createTestUserForMarker creates a user and returns their ID
func createTestUserForMarker(t *testing.T) uuid.UUID {
	var userID uuid.UUID
	err := testDB.QueryRow(`
		INSERT INTO users (email, password_hash, name, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, "markertest@example.com", "$2a$12$test", "Marker Test User", "user").Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return userID
}

func TestMarkerHandler_List_Success(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	// Create test user and markers
	userID := createTestUserForMarker(t)
	createTestMarker(t, userID)

	handler := NewMarkerHandler(testQueries, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/markers", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

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

	if response.Meta.Message != "Markers retrieved successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	data, ok := response.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", response.Data)
	}

	if len(data) != 1 {
		t.Errorf("expected 1 marker, got %d", len(data))
	}

	marker := data[0].(map[string]interface{})
	if marker["name"] != "Test Bamboo" {
		t.Errorf("expected name 'Test Bamboo', got %v", marker["name"])
	}
	if marker["short_code"] != "TEST001" {
		t.Errorf("expected short_code 'TEST001', got %v", marker["short_code"])
	}
}

func TestMarkerHandler_List_Empty(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	handler := NewMarkerHandler(testQueries, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/markers", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

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

	data, ok := response.Data.([]interface{})
	if !ok {
		t.Fatalf("expected data to be an array, got %T", response.Data)
	}

	if len(data) != 0 {
		t.Errorf("expected 0 markers, got %d", len(data))
	}
}

func TestMarkerHandler_GetByID_Success(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	userID := createTestUserForMarker(t)
	markerID := createTestMarker(t, userID)

	handler := NewMarkerHandler(testQueries, nil)

	// Create chi context with URL param
	r := chi.NewRouter()
	r.Get("/markers/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/markers/"+markerID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

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

	if response.Meta.Message != "Marker retrieved successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", response.Data)
	}

	if data["name"] != "Test Bamboo" {
		t.Errorf("expected name 'Test Bamboo', got %v", data["name"])
	}
	if data["description"] != "Test description" {
		t.Errorf("expected description 'Test description', got %v", data["description"])
	}
	if data["strain"] != "Bambusa vulgaris" {
		t.Errorf("expected strain 'Bambusa vulgaris', got %v", data["strain"])
	}
}

func TestMarkerHandler_GetByID_NotFound(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	handler := NewMarkerHandler(testQueries, nil)

	randomID := uuid.New()

	r := chi.NewRouter()
	r.Get("/markers/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/markers/"+randomID.String(), nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d: %s", http.StatusNotFound, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Marker not found" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestMarkerHandler_GetByID_InvalidID(t *testing.T) {
	handler := NewMarkerHandler(testQueries, nil)

	r := chi.NewRouter()
	r.Get("/markers/{id}", handler.GetByID)

	req := httptest.NewRequest(http.MethodGet, "/markers/invalid-uuid", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if response.Meta.Success {
		t.Error("expected success=false")
	}

	if response.Meta.Message != "Invalid marker ID" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}
}

func TestMarkerHandler_List_LightweightFields(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	userID := createTestUserForMarker(t)
	createTestMarker(t, userID)

	handler := NewMarkerHandler(testQueries, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/markers", nil)
	rr := httptest.NewRecorder()

	handler.List(rr, req)

	var response Response
	json.Unmarshal(rr.Body.Bytes(), &response)

	data := response.Data.([]interface{})
	marker := data[0].(map[string]interface{})

	// Should have lightweight fields
	expectedFields := []string{"id", "short_code", "name", "latitude", "longitude"}
	for _, field := range expectedFields {
		if _, exists := marker[field]; !exists {
			t.Errorf("expected field %s to exist in lightweight response", field)
		}
	}

	// Should NOT have full detail fields
	excludedFields := []string{"description", "strain", "quantity", "image_url", "owner_name", "owner_contact", "creator_id", "created_at", "updated_at"}
	for _, field := range excludedFields {
		if _, exists := marker[field]; exists {
			t.Errorf("field %s should not exist in lightweight response", field)
		}
	}
}

// Helper to create multipart form request for marker creation
func createMarkerFormRequest(t *testing.T, fields map[string]string) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("failed to write field %s: %v", key, err)
		}
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/markers", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// Helper to add JWT claims to request context
func addClaimsToContext(req *http.Request, userID uuid.UUID) *http.Request {
	claims := &auth.Claims{
		UserID: userID,
		Email:  "test@example.com",
		Role:   "user",
	}
	ctx := context.WithValue(req.Context(), appMiddleware.ClaimsKey, claims)
	return req.WithContext(ctx)
}

func TestMarkerHandler_Create_Success(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	// Create test user
	userID := createTestUserForMarker(t)

	handler := NewMarkerHandler(testQueries, nil)

	// Create form data with required fields
	fields := map[string]string{
		"name":        "New Bamboo Location",
		"latitude":    "-7.25000000",
		"longitude":   "110.45000000",
		"description": "A beautiful bamboo grove",
		"strain":      "Dendrocalamus asper",
		"quantity":    "100",
		"owner_name":  "John Doe",
	}

	req := createMarkerFormRequest(t, fields)
	req = addClaimsToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Error("expected success=true")
	}

	if response.Meta.Message != "Marker created successfully" {
		t.Errorf("unexpected message: %s", response.Meta.Message)
	}

	data, ok := response.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", response.Data)
	}

	// Verify returned data
	if data["name"] != "New Bamboo Location" {
		t.Errorf("expected name 'New Bamboo Location', got %v", data["name"])
	}
	if data["latitude"] != "-7.25000000" {
		t.Errorf("expected latitude '-7.25000000', got %v", data["latitude"])
	}
	if data["longitude"] != "110.45000000" {
		t.Errorf("expected longitude '110.45000000', got %v", data["longitude"])
	}

	// Verify short_code was generated (8 chars)
	shortCode, ok := data["short_code"].(string)
	if !ok || len(shortCode) != 8 {
		t.Errorf("expected short_code to be 8 characters, got %v", data["short_code"])
	}

	// Verify optional fields
	if data["description"] != "A beautiful bamboo grove" {
		t.Errorf("expected description 'A beautiful bamboo grove', got %v", data["description"])
	}
	if data["strain"] != "Dendrocalamus asper" {
		t.Errorf("expected strain 'Dendrocalamus asper', got %v", data["strain"])
	}
}

func TestMarkerHandler_Create_MinimalFields(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	userID := createTestUserForMarker(t)

	handler := NewMarkerHandler(testQueries, nil)

	// Create form data with only required fields
	fields := map[string]string{
		"name":      "Minimal Bamboo",
		"latitude":  "-7.30000000",
		"longitude": "110.50000000",
	}

	req := createMarkerFormRequest(t, fields)
	req = addClaimsToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d: %s", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var response Response
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if !response.Meta.Success {
		t.Error("expected success=true")
	}

	data := response.Data.(map[string]interface{})
	if data["name"] != "Minimal Bamboo" {
		t.Errorf("expected name 'Minimal Bamboo', got %v", data["name"])
	}

	// Optional fields should be nil
	if data["description"] != nil {
		t.Errorf("expected description to be nil, got %v", data["description"])
	}
	if data["strain"] != nil {
		t.Errorf("expected strain to be nil, got %v", data["strain"])
	}
}

func TestMarkerHandler_Create_ValidationErrors(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	userID := createTestUserForMarker(t)

	handler := NewMarkerHandler(testQueries, nil)

	// Missing required fields
	fields := map[string]string{
		"description": "Missing name, latitude, longitude",
	}

	req := createMarkerFormRequest(t, fields)
	req = addClaimsToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
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

	// Check validation errors in details
	if response.Meta.Details == nil {
		t.Fatal("expected details to be present")
	}

	if _, exists := response.Meta.Details["name"]; !exists {
		t.Error("expected validation error for 'name'")
	}
	if _, exists := response.Meta.Details["latitude"]; !exists {
		t.Error("expected validation error for 'latitude'")
	}
	if _, exists := response.Meta.Details["longitude"]; !exists {
		t.Error("expected validation error for 'longitude'")
	}
}

func TestMarkerHandler_Create_InvalidQuantity(t *testing.T) {
	cleanupMarkers(t)
	cleanupUsers(t)

	userID := createTestUserForMarker(t)

	handler := NewMarkerHandler(testQueries, nil)

	fields := map[string]string{
		"name":      "Test Bamboo",
		"latitude":  "-7.30000000",
		"longitude": "110.50000000",
		"quantity":  "not-a-number",
	}

	req := createMarkerFormRequest(t, fields)
	req = addClaimsToContext(req, userID)
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s", http.StatusBadRequest, rr.Code, rr.Body.String())
	}

	var response Response
	json.Unmarshal(rr.Body.Bytes(), &response)

	if response.Meta.Success {
		t.Error("expected success=false")
	}
}

func TestMarkerHandler_Create_Unauthorized(t *testing.T) {
	handler := NewMarkerHandler(testQueries, nil)

	fields := map[string]string{
		"name":      "Test Bamboo",
		"latitude":  "-7.30000000",
		"longitude": "110.50000000",
	}

	req := createMarkerFormRequest(t, fields)
	// Note: NOT adding claims to context
	rr := httptest.NewRecorder()

	handler.Create(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d: %s", http.StatusUnauthorized, rr.Code, rr.Body.String())
	}
}
