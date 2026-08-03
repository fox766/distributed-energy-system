package config

import (
	"os"
)

type Config struct {
	Port          string
	JWTSecret     string
	AdminUsername string
	AdminPassword string
	CryptoPath    string
}

func Load() *Config {
	cfg := &Config{
		Port:          getEnv("PORT", "8080"),
		JWTSecret:     getEnv("JWT_SECRET", "change-me-in-production"),
		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		CryptoPath:    getEnv("FABRIC_CRYPTO_PATH", "../../blockchain/network/organizations/peerOrganizations/org1.example.com"),
	}
	if cfg.JWTSecret == "change-me-in-production" {
		println("WARNING: Using default JWT_SECRET. Set JWT_SECRET env var for production.")
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
