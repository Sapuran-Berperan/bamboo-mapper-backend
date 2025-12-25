package config

import "os"

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
}

func Load() *Config {
	return &Config{
		Environment: getEnv("ENVIRONMENT", "dev"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
