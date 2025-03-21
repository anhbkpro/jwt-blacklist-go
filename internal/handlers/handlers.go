package handlers

import (
	"net/http"
	"strings"

	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
	"github.com/gin-gonic/gin"
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
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request"})
		return
	}

	// Validate input
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username and password are required"})
		return
	}

	// Check if user exists
	user, exists := h.userService.GetUserByUsername(c.Request.Context(), req.Username)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	// Verify password
	valid, err := models.VerifyPassword(req.Password, user.Password)
	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid credentials"})
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := h.jwtManager.GenerateTokens(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to generate tokens"})
		return
	}

	// Return tokens
	c.JSON(http.StatusOK, TokenResponse{
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
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// Extract refresh token from Authorization header
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization header format"})
		return
	}

	refreshTokenString := parts[1]

	// Generate a new access token
	accessToken, err := h.jwtManager.RefreshToken(refreshTokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid refresh token"})
		return
	}

	// Return the new access token
	c.JSON(http.StatusOK, TokenResponse{
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
func (h *AuthHandler) Logout(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "missing authorization header"})
		return
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "invalid authorization header format"})
		return
	}

	tokenString := parts[1]

	// Blacklist the token
	err := h.jwtManager.BlacklistToken(tokenString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to logout"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully logged out"})
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
func (h *AuthHandler) Protected(c *gin.Context) {
	// Get user claims from context (set by auth middleware)
	claims, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
		return
	}

	userClaims := claims.(*auth.JWTClaims)
	c.JSON(http.StatusOK, gin.H{
		"message": "This is a protected resource",
		"user": gin.H{
			"id":       userClaims.UserID,
			"username": userClaims.Username,
			"role":     userClaims.Role,
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
func (h *AuthHandler) AdminOnly(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "This is an admin-only resource",
	})
}
