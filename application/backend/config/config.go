package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          string
	JWTSecret     string
	AdminUsername string
	AdminPassword string
	CryptoPath    string
	// InitialBalance is credited to every newly registered user so that
	// consumers can place BUY orders (which escrow funds at creation).
	InitialBalance float64
	// DisableSchedulers turns off the background generation and auto-match
	// goroutines, which makes end-to-end balance assertions reproducible.
	DisableSchedulers bool
}

func Load() *Config {
	cfg := &Config{
		Port:              getEnv("PORT", "8080"),
		JWTSecret:         getEnv("JWT_SECRET", "change-me-in-production"),
		AdminUsername:     getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:     getEnv("ADMIN_PASSWORD", "admin123"),
		CryptoPath:        getEnv("FABRIC_CRYPTO_PATH", "../../blockchain/network/organizations/peerOrganizations/org1.example.com"),
		InitialBalance:    getEnvFloat("INITIAL_BALANCE", 1000.0),
		DisableSchedulers: getEnv("DISABLE_SCHEDULERS", "") != "",
	}
	if cfg.JWTSecret == "change-me-in-production" {
		println("WARNING: Using default JWT_SECRET. Set JWT_SECRET env var for production.")
	}
	return cfg
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 {
		return fallback
	}
	return f
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
