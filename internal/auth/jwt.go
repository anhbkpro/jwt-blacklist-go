package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/anhbkpro/jwt-blacklist-go/config"
	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenBlacklisted = errors.New("token blacklisted")
)

// JWTManager handles JWT operations

type JWTManager struct {
	config     *config.Config
	redisCache *redis.Client
}

// JWTClaims contains the claims data stored in the JWT

type JWTClaims struct {
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	TokenID   string `json:"jti"`
	TokenType string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config *config.Config, redisCache *redis.Client) *JWTManager {
	return &JWTManager{
		config:     config,
		redisCache: redisCache,
	}
}

// GenerateTokens creates new access and refresh tokens for a user
func (m *JWTManager) GenerateTokens(user *models.User) (string, string, error) {
	// Generate access token
	accessJti := generateTokenId()
	accessClaims := JWTClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		TokenID:   accessJti,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(m.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	// Generate refresh token
	refreshJti := generateTokenId()
	refreshClaims := JWTClaims{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		TokenID:   refreshJti,
		TokenType: "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(m.config.JWTSecret))
	if err != nil {
		return "", "", err
	}

	return accessTokenString, refreshTokenString, nil
}

// VerifyToken validates the token and returns the claims
func (m *JWTManager) VerifyToken(tokenString string) (*JWTClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(m.config.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}

		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is blacklisted
	isBlacklisted, err := m.IsTokenBlacklisted(claims.TokenID)
	if err != nil {
		return nil, err
	}
	if isBlacklisted {
		return nil, ErrTokenBlacklisted
	}

	// Return the claims
	return claims, nil
}

// RefreshToken creates a new access token from a valid refresh token
func (m *JWTManager) RefreshToken(refreshTokenString string) (string, error) {
	claims, err := m.VerifyToken(refreshTokenString)
	if err != nil {
		return "", err
	}

	// Ensure this is a refresh token
	if claims.TokenType != "refresh" {
		return "", errors.New("not a refresh token")
	}

	// Create a new access token
	accessJti := generateTokenId()
	accessClaims := JWTClaims{
		UserID:    claims.UserID,
		Username:  claims.Username,
		Role:      claims.Role,
		TokenID:   accessJti,
		TokenType: "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenExpiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(m.config.JWTSecret))
	if err != nil {
		return "", err
	}

	return accessTokenString, nil
}

// BlacklistToken adds a token to the blacklist
func (m *JWTManager) BlacklistToken(tokenString string) error {
	// Parse the token (without full verification)
	token, _ := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.config.JWTSecret), nil
	})

	// Even if token is invalid, we try to extract the claims
	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return errors.New("could not parse token claims")
	}

	// Get the token ID and expiration time
	jti := claims.TokenID
	exp := claims.ExpiresAt

	// Calculate time until token expiration
	var ttl time.Duration
	if exp != nil {
		ttl = time.Until(exp.Time)
		if ttl < 0 {
			// Token already expired, no need to blacklist
			return nil
		}
	} else {
		// If expiration can't be determined, set a default expiration
		ttl = 24 * time.Hour
	}

	// Store token ID in Redis with TTL
	ctx := context.Background()
	key := fmt.Sprintf("blacklist:%s", jti)
	err := m.redisCache.Set(ctx, key, "1", ttl).Err()
	log.Printf("--- Blacklisted token %s with TTL %s, key %s", jti, ttl, key)
	return err
}

// IsTokenBlacklisted checks if a token is blacklisted
func (m *JWTManager) IsTokenBlacklisted(jti string) (bool, error) {
	ctx := context.Background()
	result, err := m.redisCache.Exists(ctx, fmt.Sprintf("blacklist:%s", jti)).Result()

	if err != nil {
		return false, err
	}

	return result > 0, nil
}

// Helper function to generate a unique token ID
func generateTokenId() string {
	// Create a unique token ID by combining timestamp and random values
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 16)
	randomBytes := make([]byte, 8)
	_, _ = time.Now().UnixNano(), randomBytes

	hash := sha256.Sum256([]byte(timestamp + string(randomBytes)))
	return hex.EncodeToString(hash[:])
}
