package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware is a middleware that validates JWT tokens
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// Authenticate middleware for Gin
func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization header format"})
			return
		}

		tokenString := parts[1]

		claims, err := m.jwtManager.VerifyToken(tokenString)
		if err != nil {
			var message string
			switch {
			case errors.Is(err, auth.ErrInvalidToken):
				message = "invalid token"
			case errors.Is(err, auth.ErrTokenExpired):
				message = "token expired"
			case errors.Is(err, auth.ErrTokenBlacklisted):
				message = "token has been revoked"
			default:
				message = err.Error()
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": message})
			return
		}

		if claims.TokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid token type"})
			return
		}

		c.Set("user", claims)
		c.Next()
	}
}

// RequireRole middleware for Gin
func (m *AuthMiddleware) RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userClaims, exists := c.Get("user")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "user not authenticated"})
			return
		}

		claims := userClaims.(*auth.JWTClaims)
		if claims.Role != role {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
