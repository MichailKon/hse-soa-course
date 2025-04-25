package main

import (
	"fmt"
	"log"
	"os"
	"social-network/common/kafka"
	"social-network/user-service/handlers"
	"social-network/user-service/middleware"
	"social-network/user-service/models"
	"social-network/user-service/repositories"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s dbname=%s sslmode=disable password=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PASSWORD"))

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate table User: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)
	producer := kafka.NewProducer()
	if producer == nil {
		log.Fatal("Failed to create producer")
	}

	jwtKey := os.Getenv("JWT_SECRET")
	if jwtKey == "" {
		jwtKey = "JWT_SECRET" // DEV ONLY
	}

	userHandler := handlers.NewUserHandler(userRepo, jwtKey, producer)

	router := gin.Default()

	router.POST("/api/auth/register", userHandler.Register)
	router.POST("/api/auth/login", userHandler.Login)

	auth := router.Group("/api/users")
	auth.Use(middleware.AuthMiddleware(jwtKey))
	{
		auth.GET("/profile", userHandler.GetProfile)
		auth.PUT("/profile", userHandler.UpdateProfile)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("User service starting on port %s", port)
	if err = router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
