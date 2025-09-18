package config

import (
	"os"
)

type Config struct {
	DatabaseURL   string
	SteamAPIKey   string
	BuffAPIKey    string
	YoupinAPIKey  string
	JWTSecret     string
	Port          string
	Environment   string
}

func Load() *Config {
	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "csgo_trader.db"),
		SteamAPIKey:   getEnv("STEAM_API_KEY", ""),
		BuffAPIKey:    getEnv("BUFF_API_KEY", ""),
		YoupinAPIKey:  getEnv("YOUPIN_API_KEY", ""),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key"),
		Port:          getEnv("PORT", "8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}