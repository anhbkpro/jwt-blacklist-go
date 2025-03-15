package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/anhbkpro/jwt-blacklist-go/config"
	"github.com/anhbkpro/jwt-blacklist-go/internal/auth"
	"github.com/anhbkpro/jwt-blacklist-go/internal/handlers"
	"github.com/anhbkpro/jwt-blacklist-go/internal/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg := config.NewConfig()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := redisClient.Ping(ctx).Result()
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
	authHandler := handlers.NewAuthHandler(jwtManager)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

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
		if err := e.Start(":8088"); err != nil && err != http.ErrServerClosed {
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
