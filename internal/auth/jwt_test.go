package auth

import (
	"context"
	"testing"
	"time"

	"github.com/anhbkpro/jwt-blacklist-go/config"
	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRedisClient creates a test Redis client using a real Redis instance
// For proper testing, you should have Redis running on localhost:6379
// In a CI environment, you might want to use a Redis mock instead
func mockRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       15, // Use DB 15 for testing to avoid conflicts
	})
}

// cleanupRedis clears the test Redis database
func cleanupRedis(client *redis.Client) {
	client.FlushDB(context.Background())
}

func TestMultiDeviceLogout(t *testing.T) {
	// Initialize Redis client
	redisClient := mockRedisClient()
	defer cleanupRedis(redisClient)

	// Create a test context
	ctx := context.Background()

	// Ping Redis to make sure it's available
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// Create test configuration
	cfg := &config.Config{
		JWTSecret:              "test-secret-key",
		AccessTokenExpiration:  15 * time.Minute,
		RefreshTokenExpiration: 7 * 24 * time.Hour,
	}

	// Create JWT manager
	jwtManager := NewJWTManager(cfg, redisClient)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}

	// Test case: Multi-device scenario
	t.Run("MultiDeviceLogout", func(t *testing.T) {
		// Step 1: Generate tokens for "iPhone" (first device)
		iPhoneAccessToken, _, err := jwtManager.GenerateTokens(user)
		require.NoError(t, err)
		require.NotEmpty(t, iPhoneAccessToken)

		// Verify the iPhone access token is valid
		iPhoneAccessClaims, err := jwtManager.VerifyToken(iPhoneAccessToken)
		require.NoError(t, err)
		require.NotNil(t, iPhoneAccessClaims)

		// Store the token ID for later reference
		iPhoneTokenID := iPhoneAccessClaims.TokenID

		// Step 2: Generate tokens for "iPad" (second device)
		iPadAccessToken, _, err := jwtManager.GenerateTokens(user)
		require.NoError(t, err)
		require.NotEmpty(t, iPadAccessToken)

		// Verify the iPad access token is valid
		iPadAccessClaims, err := jwtManager.VerifyToken(iPadAccessToken)
		require.NoError(t, err)
		require.NotNil(t, iPadAccessClaims)

		// Store the token ID for later reference
		iPadTokenID := iPadAccessClaims.TokenID

		// Ensure the two tokens have different JTIs
		assert.NotEqual(t, iPhoneTokenID, iPadTokenID, "Tokens should have unique JTIs")

		// Step 3: Logout from iPad (blacklist its token)
		err = jwtManager.BlacklistToken(iPadAccessToken)
		require.NoError(t, err)

		// Verify iPad token is now blacklisted
		isBlacklisted, err := jwtManager.IsTokenBlacklisted(iPadTokenID)
		require.NoError(t, err)
		assert.True(t, isBlacklisted, "iPad token should be blacklisted")

		// Verify iPhone token is NOT blacklisted
		isBlacklisted, err = jwtManager.IsTokenBlacklisted(iPhoneTokenID)
		require.NoError(t, err)
		assert.False(t, isBlacklisted, "iPhone token should not be blacklisted")

		// Step 4: Try to verify both tokens
		// iPad token should fail verification
		_, err = jwtManager.VerifyToken(iPadAccessToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenBlacklisted, err)

		// iPhone token should still pass verification
		_, err = jwtManager.VerifyToken(iPhoneAccessToken)
		assert.NoError(t, err)

		// Step 5: Logout from iPhone too
		err = jwtManager.BlacklistToken(iPhoneAccessToken)
		require.NoError(t, err)

		// Verify iPhone token is now blacklisted
		isBlacklisted, err = jwtManager.IsTokenBlacklisted(iPhoneTokenID)
		require.NoError(t, err)
		assert.True(t, isBlacklisted, "iPhone token should now be blacklisted")

		// Both tokens should now fail verification
		_, err = jwtManager.VerifyToken(iPhoneAccessToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenBlacklisted, err)
	})
}

func TestTokenExpirationInBlacklist(t *testing.T) {
	// Initialize Redis client
	redisClient := mockRedisClient()
	defer cleanupRedis(redisClient)

	// Create a test context
	ctx := context.Background()

	// Ping Redis to make sure it's available
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		t.Skip("Redis not available, skipping test")
	}

	// Create test configuration with very short token expiration for testing
	cfg := &config.Config{
		JWTSecret:              "test-secret-key",
		AccessTokenExpiration:  2 * time.Second, // Very short for testing
		RefreshTokenExpiration: 5 * time.Second,
	}

	// Create JWT manager
	jwtManager := NewJWTManager(cfg, redisClient)

	// Create a test user
	user := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}

	// Test case: Token expiration in blacklist
	t.Run("BlacklistedTokenExpiration", func(t *testing.T) {
		// Generate tokens
		accessToken, _, err := jwtManager.GenerateTokens(user)
		require.NoError(t, err)

		// Verify token
		accessClaims, err := jwtManager.VerifyToken(accessToken)
		require.NoError(t, err)

		// Blacklist the token
		err = jwtManager.BlacklistToken(accessToken)
		require.NoError(t, err)

		// Verify token is blacklisted
		isBlacklisted, err := jwtManager.IsTokenBlacklisted(accessClaims.TokenID)
		require.NoError(t, err)
		assert.True(t, isBlacklisted)

		// Check that token verification fails
		_, err = jwtManager.VerifyToken(accessToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenBlacklisted, err)

		// Wait for token to expire in both JWT and Redis
		time.Sleep(3 * time.Second)

		// Blacklist entry should be automatically removed by Redis
		isBlacklisted, err = jwtManager.IsTokenBlacklisted(accessClaims.TokenID)
		require.NoError(t, err)
		assert.False(t, isBlacklisted, "Blacklist entry should be automatically removed")

		// Token verification should still fail, but now due to expiration
		_, err = jwtManager.VerifyToken(accessToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenExpired, err)
	})
}
