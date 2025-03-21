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
	jwtManager  *auth.JWTManager
	userService *models.UserService
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(jwtManager *auth.JWTManager, userService *models.UserService) *AuthHandler {
	return &AuthHandler{
		jwtManager:  jwtManager,
		userService: userService,
	}
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" example:"admin"`
	Password string `json:"password" example:"admin123"`
}

// TokenResponse represents the response for token requests
type TokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int    `json:"expires_in" example:"900"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Message string `json:"message" example:"Invalid credentials"`
}

// Login handles the login request
// @Summary Login to the system
// @Description Authenticate user and get JWT tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login request"
// @Success 200 {object} TokenResponse "Successful login"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 401 {object} ErrorResponse "Invalid credentials"
// @Router /auth/login [post]
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
	user, exists := h.userService.GetUserByUsername(c.Request().Context(), req.Username)
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
// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TokenResponse "New access token"
// @Failure 401 {object} ErrorResponse "Invalid refresh token"
// @Router /auth/refresh [post]
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
// @Summary Logout from the system
// @Description Revoke the current token
// @Tags auth
// @Security BearerAuth
// @Success 200 {object} map[string]string "Successfully logged out"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /auth/logout [post]
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
// @Summary Get protected resource
// @Description Access a protected resource requiring authentication
// @Tags protected
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Protected resource"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Router /protected [get]
func (h *AuthHandler) Protected(c echo.Context) error {
	// Get user claims from context (set by auth middleware)
	claims, ok := c.Get("user").(*auth.JWTClaims)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "This is a protected resource",
		"user": map[string]interface{}{
			"id":       claims.UserID,
			"username": claims.Username,
			"role":     claims.Role,
		},
	})
}

// AdminOnly is a handler for admin-only resources
// @Summary Get admin resource
// @Description Access an admin-only resource
// @Tags admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string "Admin resource"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 403 {object} ErrorResponse "Forbidden"
// @Router /admin/dashboard [get]
func (h *AuthHandler) AdminOnly(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "This is an admin-only resource",
	})
}
