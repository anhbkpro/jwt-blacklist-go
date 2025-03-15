# JWT Token Blacklisting Implementation

## Overview

This document describes the token blacklisting mechanism used in our JWT authentication system. The implementation uses Redis to efficiently store and manage blacklisted tokens without keeping a permanent record of all tokens.

## How It Works

Our token blacklisting system uses Redis with automatic expiration (TTL) to maintain a list of invalidated tokens. This approach offers several advantages:

1. Minimal storage requirements - only blacklisted tokens are stored
2. Automatic cleanup - expired entries are removed without manual intervention
3. Fast lookup - O(1) complexity for checking token status
4. Scalable - works efficiently even with large numbers of tokens

## Token Status Logic

The system handles tokens based on these rules:

* **Tokens not in the blacklist**: allowed access
* **Tokens in the blacklist**: denied access
* **Tokens whose blacklist entry expired**: allowed access (because by then the token itself has expired too)

## Implementation Details

When a token is blacklisted (e.g., during logout):

```go
// BlacklistToken adds a token to the blacklist
func (m *JWTManager) BlacklistToken(tokenString string) error {
    // Parse the token (without full verification)
    token, _ := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(m.config.JWTSecret), nil
    })

    // Extract the claims
    claims, ok := token.Claims.(*JWTClaims)
    if !ok {
        return errors.New("could not parse token claims")
    }

    // Get the token ID and expiration time
    jti := claims.TokenID
    exp := claims.ExpiresAt

    // Calculate time until token expiration
    var ttl time.Duration
    if exp != nil {
        ttl = time.Until(exp.Time)
        if ttl <= 0 {
            // Token already expired, no need to blacklist
            return nil
        }
    } else {
        // If expiration can't be determined, set a default expiration
        ttl = 24 * time.Hour
    }

    // Store token ID in Redis with TTL
    ctx := context.Background()
    err := m.redisCache.Set(ctx, fmt.Sprintf("blacklist:%s", jti), "1", ttl).Err()

    return err
}
```

When verifying a token:

```go
// VerifyToken validates the token and returns the claims
func (m *JWTManager) VerifyToken(tokenString string) (*JWTClaims, error) {
    // Parse the token
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        // Validate the signing method
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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

    return claims, nil
}
```

Checking if a token is blacklisted:

```go
// IsTokenBlacklisted checks if a token is blacklisted
func (m *JWTManager) IsTokenBlacklisted(jti string) (bool, error) {
    ctx := context.Background()
    result, err := m.redisCache.Exists(ctx, fmt.Sprintf("blacklist:%s", jti)).Result()

    if err != nil {
        return false, err
    }

    return result > 0, nil
}
```

## Redis Commands for Monitoring

To check blacklisted tokens in Redis CLI:

```
# List all blacklisted tokens
KEYS blacklist:*

# Check if a specific token is blacklisted
EXISTS blacklist:<token-id>

# Check how long until a blacklisted token expires
TTL blacklist:<token-id>
```

## Benefits of This Approach

1. **Efficiency**: Only stores the minimal information needed (token ID)
2. **Self-maintaining**: Redis automatically removes expired entries
3. **Performance**: Redis provides fast lookups for token validation
4. **Simplicity**: No need for complex database schemas or maintenance tasks

## Unit Tests

The system includes comprehensive unit tests to verify the token blacklisting functionality, particularly for multi-device scenarios.

### Multi-Device Test

This test verifies that logging out from one device doesn't affect sessions on other devices:

```go
func TestMultiDeviceLogout(t *testing.T) {
    // ...test setup...

    // Step 1: Generate tokens for "iPhone" (first device)
    iPhoneAccessToken, _, err := jwtManager.GenerateTokens(user)

    // Step 2: Generate tokens for "iPad" (second device)
    iPadAccessToken, _, err := jwtManager.GenerateTokens(user)

    // Ensure the two tokens have different JTIs
    assert.NotEqual(t, iPhoneTokenID, iPadTokenID, "Tokens should have unique JTIs")

    // Step 3: Logout from iPad (blacklist its token)
    err = jwtManager.BlacklistToken(iPadAccessToken)

    // Verify iPad token is now blacklisted
    isBlacklisted, err := jwtManager.IsTokenBlacklisted(iPadTokenID)
    assert.True(t, isBlacklisted, "iPad token should be blacklisted")

    // Verify iPhone token is NOT blacklisted
    isBlacklisted, err = jwtManager.IsTokenBlacklisted(iPhoneTokenID)
    assert.False(t, isBlacklisted, "iPhone token should not be blacklisted")

    // Step 4: Try to verify both tokens
    // iPad token should fail verification
    _, err = jwtManager.VerifyToken(iPadAccessToken)
    assert.Error(t, err)
    assert.Equal(t, ErrTokenBlacklisted, err)

    // iPhone token should still pass verification
    _, err = jwtManager.VerifyToken(iPhoneAccessToken)
    assert.NoError(t, err)
}
```

### Token Expiration Test

This test verifies that blacklisted tokens are automatically removed from Redis when they expire:

```go
func TestTokenExpirationInBlacklist(t *testing.T) {
    // ...test setup with short-lived tokens...

    // Blacklist the token
    err = jwtManager.BlacklistToken(accessToken)

    // Verify token is blacklisted
    isBlacklisted, err := jwtManager.IsTokenBlacklisted(accessClaims.TokenID)
    assert.True(t, isBlacklisted)

    // Wait for token to expire in both JWT and Redis
    time.Sleep(3 * time.Second)

    // Blacklist entry should be automatically removed by Redis
    isBlacklisted, err = jwtManager.IsTokenBlacklisted(accessClaims.TokenID)
    assert.False(t, isBlacklisted, "Blacklist entry should be automatically removed")

    // Token verification should still fail, but now due to expiration
    _, err = jwtManager.VerifyToken(accessToken)
    assert.Error(t, err)
    assert.Equal(t, ErrTokenExpired, err)
}
```

### Running the Tests

Use the provided Makefile commands to run the tests:

```bash
# Run all tests
make test

# Run only the multi-device tests
make test-multidevice

# Run token expiration tests
make test-expiration
```

Note: The tests require a running Redis instance on localhost:6379.
