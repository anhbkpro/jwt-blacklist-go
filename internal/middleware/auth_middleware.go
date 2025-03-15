package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/labstack/echo/v4"
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

// Authenticate is a middleware function that authenticates requests
func (m *AuthMiddleware) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get the token from the request
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
		}

		tokenString := parts[1]

		// Verify the token
		claims, err := m.jwtManager.VerifyToken(tokenString)
		if err != nil {
			if errors.Is(err, auth.ErrInvalidToken) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}
			if errors.Is(err, auth.ErrTokenExpired) {
				return echo.NewHTTPError(http.StatusUnauthorized, "token expired")
			}
			if errors.Is(err, auth.ErrTokenBlacklisted) {
				return echo.NewHTTPError(http.StatusUnauthorized, "token has been revoked")
			}
			return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
		}

		// Ensure this is an access token
		if claims.TokenType != "access" {
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token type")
		}

		// Store the claims in the context
		c.Set("user", claims)

		// Call the next handler
		return next(c)
	}
}

// RequireRole middleware ensures that a user has a specific role
func (m *AuthMiddleware) RequireRole(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the user claims from context (set by the Authenticate middleware)
			userClaims, ok := c.Get("user").(*auth.JWTClaims)
			if !ok {
				return echo.NewHTTPError(http.StatusUnauthorized, "user not authenticated")
			}

			// Check if the user has the required role
			if userClaims.Role != role {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}

			return next(c)
		}
	}
}
