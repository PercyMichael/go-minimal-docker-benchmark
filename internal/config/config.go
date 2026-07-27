package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabaseURL  string
	TargetDBName string
	JWTSecret    string
}

func Load() *Config {
	// Automatically load local .env file if present (ignored if not found)
	_ = godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	targetDBName := os.Getenv("POSTGRES_DB")
	if targetDBName == "" {
		targetDBName = "zennotes"
	}

	dbURL := os.Getenv("DATABASE_URL")

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-key-change-in-production"
	}

	return &Config{
		Port:         port,
		DatabaseURL:  dbURL,
		TargetDBName: targetDBName,
		JWTSecret:    jwtSecret,
	}
}
