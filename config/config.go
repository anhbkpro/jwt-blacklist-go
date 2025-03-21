package config

import (
	"os"
	"strconv"
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
	DB                     *DBConfig
}

// DBConfig holds database configuration
type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int // minutes
}

// NewConfig creates a new Config struct with values from environment or defaults
func NewConfig() *Config {
	jwtSecret := getEnv("JWT_SECRET", "your-super-secret-key-change-in-production")
	accessExpStr := getEnv("ACCESS_TOKEN_EXPIRATION", "15m")
	refreshExpStr := getEnv("REFRESH_TOKEN_EXPIRATION", "7d")

	accessExp, _ := time.ParseDuration(accessExpStr)
	refreshExp, _ := time.ParseDuration(refreshExpStr)

	// Parse database configuration
	dbPort, _ := strconv.Atoi(getEnv("DB_PORT", "5432"))
	maxOpenConns, _ := strconv.Atoi(getEnv("DB_MAX_OPEN_CONNS", "10"))
	maxIdleConns, _ := strconv.Atoi(getEnv("DB_MAX_IDLE_CONNS", "5"))
	connMaxLifetime, _ := strconv.Atoi(getEnv("DB_CONN_MAX_LIFETIME", "30"))

	dbConfig := &DBConfig{
		Host:            getEnv("DB_HOST", ""),
		Port:            dbPort,
		User:            getEnv("DB_USER", ""),
		Password:        getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", ""),
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
	}

	return &Config{
		JWTSecret:              jwtSecret,
		AccessTokenExpiration:  accessExp,
		RefreshTokenExpiration: refreshExp,
		RedisAddr:              getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          getEnv("REDIS_PASSWORD", ""),
		RedisDB:                0,
		DB:                     dbConfig,
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
