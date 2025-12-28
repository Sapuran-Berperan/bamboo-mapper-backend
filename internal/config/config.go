package config

import (
	"os"
	"time"
)

type Config struct {
	Environment           string
	Port                  string
	DatabaseURL           string
	JWTSecret             string
	AccessTokenExpiry     time.Duration
	RefreshTokenExpiry    time.Duration
	GDriveCredentialsPath string
	GDriveTokenPath       string
	GDriveFolderID        string
}

func Load() *Config {
	env := getEnv("ENVIRONMENT", "dev")

	var dbURL string
	if env == "dev" {
		dbURL = getEnv("DATABASE_URL_DEV", "")
	} else {
		dbURL = getEnv("DATABASE_URL", "")
	}

	accessExpiry := parseDuration(getEnv("ACCESS_TOKEN_EXPIRY", "15m"), 15*time.Minute)
	refreshExpiry := parseDuration(getEnv("REFRESH_TOKEN_EXPIRY", "168h"), 7*24*time.Hour)

	return &Config{
		Environment:           env,
		Port:                  getEnv("PORT", "8080"),
		DatabaseURL:           dbURL,
		JWTSecret:             getEnv("JWT_SECRET", ""),
		AccessTokenExpiry:     accessExpiry,
		RefreshTokenExpiry:    refreshExpiry,
		GDriveCredentialsPath: getEnv("GDRIVE_CREDENTIALS_PATH", ""),
		GDriveTokenPath:       getEnv("GDRIVE_TOKEN_PATH", ""),
		GDriveFolderID:        getEnv("GDRIVE_FOLDER_ID", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func parseDuration(s string, defaultValue time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return defaultValue
	}
	return d
}
