package config

import "os"

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
}

func Load() *Config {
	env := getEnv("ENVIRONMENT", "dev")

	var dbURL string
	if env == "dev" {
		dbURL = getEnv("DATABASE_URL_DEV", "")
	} else {
		dbURL = getEnv("DATABASE_URL", "")
	}

	return &Config{
		Environment: env,
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: dbURL,
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
