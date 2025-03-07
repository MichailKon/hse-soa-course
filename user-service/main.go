package main

import (
	"log"
	"os"
	"user-service/handlers"
	"user-service/middleware"
	"user-service/models"
	"user-service/repositories"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=postgres dbname=users port=5432 sslmode=disable"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if err = db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("Failed to migrate table User: %v", err)
	}

	userRepo := repositories.NewUserRepository(db)

	jwtKey := os.Getenv("JWT_SECRET_KEY")
	if jwtKey == "" {
		jwtKey = "default_secret_key" // DEV ONLY
	}

	userHandler := handlers.NewUserHandler(userRepo, jwtKey)

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
