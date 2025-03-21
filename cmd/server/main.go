package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/anhbkpro/jwt-blacklist-go/config"
	_ "github.com/anhbkpro/jwt-blacklist-go/docs" // This is required for swagger
	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/anhbkpro/jwt-blacklist-go/internal/db"
	"github.com/anhbkpro/jwt-blacklist-go/internal/handlers"
	"github.com/anhbkpro/jwt-blacklist-go/internal/middleware"
	"github.com/anhbkpro/jwt-blacklist-go/internal/models"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
)

// @title JWT Blacklisting API
// @version 1.0
// @description API for JWT authentication with token blacklisting
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.yourcompany.com/support
// @contact.email support@yourcompany.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and the JWT token.
func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize PostgreSQL database
	var userRepo models.UserRepository
	var postgres *db.PostgresDB
	var err error

	// Check if DB configuration is provided
	if cfg.DB.Host != "" {
		postgres, err = db.NewPostgresDB(cfg.DB)
		if err != nil {
			log.Printf("Warning: PostgreSQL connection failed: %v", err)
			log.Println("Continuing with in-memory user store")
		} else {
			defer postgres.Close()

			// Run migrations
			migrationsPath := filepath.Join(".", "migrations")
			if err := postgres.RunMigrations(migrationsPath); err != nil {
				log.Printf("Warning: Database migrations failed: %v", err)
			}

			// Initialize user repository with PostgreSQL
			userRepo = models.NewPostgresUserRepository(postgres.DB)
			log.Println("Using PostgreSQL user repository")
		}
	}

	// Fallback to in-memory repository if PostgreSQL is not available
	if userRepo == nil {
		log.Println("Using in-memory user repository")
		userRepo = &models.InMemoryUserRepository{Users: models.DefaultUsers}
	}

	// Create user service
	userService := models.NewUserService(userRepo)

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
		log.Println("Continuing without Redis (token blacklisting will not work)")
	} else {
		log.Println("Connected to Redis successfully")
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManager(cfg, redisClient)

	// Initialize auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(jwtManager, userService)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	// Swagger documentation
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "JWT Blacklisting Demo API")
	})

	// Auth routes (public)
	e.POST("/api/auth/login", authHandler.Login)
	e.POST("/api/auth/refresh", authHandler.RefreshToken)
	e.POST("/api/auth/logout", authHandler.Logout)

	// Protected routes
	protected := e.Group("/api")
	protected.Use(authMiddleware.Authenticate)

	protected.GET("/protected", authHandler.Protected)

	// Admin-only routes
	admin := protected.Group("/admin")
	admin.Use(authMiddleware.RequireRole("admin"))
	admin.GET("/dashboard", authHandler.AdminOnly)

	// Start server
	go func() {
		if err := e.Start(":8080"); err != nil && err != http.ErrServerClosed {
			// log the error
			log.Printf("Error starting server: %v", err)
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
