package handlers

import (
	"net/http"
	"strings"

	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	jwtManager *auth.JWTManager
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		jwtManager: jwtManager,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// TokenResponse represents the response for token requests
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message"`
}

// Login handles the login request
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "username and password are required")
	}

	// Check if user exists
	user, exists := models.GetUserByUsername(req.Username)
	if !exists {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Verify password
	valid, err := models.VerifyPassword(req.Password, user.Password)
	if err != nil || !valid {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid credentials")
	}

	// Generate tokens
	accessToken, refreshToken, err := h.jwtManager.GenerateTokens(user)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to generate tokens")
	}

	// Return tokens
	return c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes in seconds
	})
}

// RefreshToken handles token refresh requests
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	// Extract refresh token from Authorization header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
	}

	refreshTokenString := parts[1]

	// Generate a new access token
	accessToken, err := h.jwtManager.RefreshToken(refreshTokenString)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid refresh token")
	}

	// Return the new access token
	return c.JSON(http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   900, // 15 minutes in seconds
	})
}

// Logout handles logout requests (token revocation)
func (h *AuthHandler) Logout(c echo.Context) error {
	// Extract token from Authorization header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
	}

	tokenString := parts[1]

	// Blacklist the token
	err := h.jwtManager.BlacklistToken(tokenString)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to logout")
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "successfully logged out"})
}

// Protected is a handler for a protected resource
func (h *AuthHandler) Protected(c echo.Context) error {
	// Get user claims from context (set by auth middleware)
	claims, ok := c.Get("user").(*auth.JWTClaims)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "This is a protected resource",
		"user": map[string]interface{}{
			"id":       claims.UserId,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}

// AdminOnly is a handler for admin-only resources
func (h *AuthHandler) AdminOnly(c echo.Context) error {
	// Get user claims from context (set by auth middleware)
	claims, ok := c.Get("user").(*auth.JWTClaims)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	// Check if the user has the required role
	if claims.Role != "admin" {
		return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "This is an admin-only resource",
	})
}
