package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	JWTSecret  string
	ServerPort string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no se encontró archivo .env, se usarán variables de entorno del sistema")
	}

	cfg := &Config{
		DBHost:     getEnv("DB_HOST", ""),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", ""),
		DBPassword: getEnv("DB_PASSWORD", ""),
		DBName:     getEnv("DB_NAME", "defaultdb"),
		DBSSLMode:  getEnv("DB_SSLMODE", "require"),
		JWTSecret:  getEnv("JWT_SECRET", ""),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}

	cfg.validate()
	return cfg
}

func (c *Config) validate() {
	missing := []string{}
	if c.DBHost == "" {
		missing = append(missing, "DB_HOST")
	}
	if c.DBUser == "" {
		missing = append(missing, "DB_USER")
	}
	if c.DBPassword == "" {
		missing = append(missing, "DB_PASSWORD")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	}
	if len(missing) > 0 {
		log.Fatalf("faltan variables de entorno obligatorias en .env: %v", missing)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}