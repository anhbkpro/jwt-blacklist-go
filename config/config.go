package config

import (
	"os"
	"time"
)

// Config holds all configuration for the application

type Config struct {
	JWTSecret              string
	AccessTokenExpiration  time.Duration
	RefreshTokenExpiration time.Duration
	RedisAddr              string
	RedisPassword          string
	RedisDB                int
}

// NewConfig creates a new Config struct with values from environment or defaults
func NewConfig() *Config {
	jwtSecret := getEnv("JWT_SECRET", "super-secret")
	accessExpStr := getEnv("ACCESS_TOKEN_EXPIRATION", "15m")
	refreshExpStr := getEnv("REFRESH_TOKEN_EXPIRATION", "7d")

	accessExp, _ := time.ParseDuration(accessExpStr)
	refreshExp, _ := time.ParseDuration(refreshExpStr)

	redisAddress := getEnv("REDIS_ADDRESS", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	return &Config{
		JWTSecret:              jwtSecret,
		AccessTokenExpiration:  accessExp,
		RefreshTokenExpiration: refreshExp,
		RedisAddr:              redisAddress,
		RedisPassword:          redisPassword,
		RedisDB:                redisDB,
	}
}

// Helper function to get environment variable with a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
