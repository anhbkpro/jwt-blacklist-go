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
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

	// Initialize Gin instead of Echo
	r := gin.Default() // This includes Logger and Recovery middleware

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "JWT Blacklisting Demo API")
	})

	// Auth routes (public)
	r.POST("/api/auth/login", authHandler.Login)
	r.POST("/api/auth/refresh", authHandler.RefreshToken)
	r.POST("/api/auth/logout", authHandler.Logout)

	// Protected routes group
	protected := r.Group("/api")
	protected.Use(authMiddleware.Authenticate())

	protected.GET("/protected", authHandler.Protected)

	// Admin-only routes
	admin := protected.Group("/admin")
	admin.Use(authMiddleware.RequireRole("admin"))
	admin.GET("/dashboard", authHandler.AdminOnly)

	// Create http.Server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Error starting server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}
