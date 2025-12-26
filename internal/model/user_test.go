package model

import (
	"testing"
)

func TestRegisterRequest_Validate(t *testing.T) {
	tests := []struct {
		name           string
		request        RegisterRequest
		expectedErrors map[string]string
	}{
		{
			name: "valid request",
			request: RegisterRequest{
				Email:    "test@example.com",
				Name:     "John Doe",
				Password: "password123",
			},
			expectedErrors: map[string]string{},
		},
		{
			name: "empty email",
			request: RegisterRequest{
				Email:    "",
				Name:     "John Doe",
				Password: "password123",
			},
			expectedErrors: map[string]string{
				"email": "email is required",
			},
		},
		{
			name: "invalid email format - no @",
			request: RegisterRequest{
				Email:    "testexample.com",
				Name:     "John Doe",
				Password: "password123",
			},
			expectedErrors: map[string]string{
				"email": "invalid email format",
			},
		},
		{
			name: "invalid email format - no domain",
			request: RegisterRequest{
				Email:    "test@",
				Name:     "John Doe",
				Password: "password123",
			},
			expectedErrors: map[string]string{
				"email": "invalid email format",
			},
		},
		{
			name: "empty name",
			request: RegisterRequest{
				Email:    "test@example.com",
				Name:     "",
				Password: "password123",
			},
			expectedErrors: map[string]string{
				"name": "name is required",
			},
		},
		{
			name: "name too long",
			request: RegisterRequest{
				Email:    "test@example.com",
				Name:     string(make([]byte, 101)),
				Password: "password123",
			},
			expectedErrors: map[string]string{
				"name": "name must be 100 characters or less",
			},
		},
		{
			name: "empty password",
			request: RegisterRequest{
				Email:    "test@example.com",
				Name:     "John Doe",
				Password: "",
			},
			expectedErrors: map[string]string{
				"password": "password is required",
			},
		},
		{
			name: "password too short",
			request: RegisterRequest{
				Email:    "test@example.com",
				Name:     "John Doe",
				Password: "short",
			},
			expectedErrors: map[string]string{
				"password": "password must be at least 8 characters",
			},
		},
		{
			name: "multiple validation errors",
			request: RegisterRequest{
				Email:    "invalid",
				Name:     "",
				Password: "short",
			},
			expectedErrors: map[string]string{
				"email":    "invalid email format",
				"name":     "name is required",
				"password": "password must be at least 8 characters",
			},
		},
		{
			name: "trims whitespace from email and name",
			request: RegisterRequest{
				Email:    "  test@example.com  ",
				Name:     "  John Doe  ",
				Password: "password123",
			},
			expectedErrors: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := tt.request.Validate()

			if len(errors) != len(tt.expectedErrors) {
				t.Errorf("expected %d errors, got %d: %v", len(tt.expectedErrors), len(errors), errors)
				return
			}

			for field, expectedMsg := range tt.expectedErrors {
				if errors[field] != expectedMsg {
					t.Errorf("expected error for %s: %q, got %q", field, expectedMsg, errors[field])
				}
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.id", true},
		{"user+tag@example.org", true},
		{"", false},
		{"@example.com", false},
		{"test@", false},
		{"test@.", false},
		{"test@example", false},
		{"testexample.com", false},
		{"test@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := isValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("isValidEmail(%q) = %v, expected %v", tt.email, result, tt.expected)
			}
		})
	}
}
