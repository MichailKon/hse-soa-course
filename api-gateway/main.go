package main

import (
	"api-gateway/middleware"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	router := gin.Default()

	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://localhost:8081"
	}

	router.POST("/api/auth/register", proxyHandler(userServiceURL+"/api/auth/register"))
	router.POST("/api/auth/login", proxyHandler(userServiceURL+"/api/auth/login"))

	auth := router.Group("/api/users")
	auth.Use(middleware.AuthMiddleware(userServiceURL))
	{
		auth.GET("/profile", proxyHandler(userServiceURL+"/api/users/profile"))
		auth.PUT("/profile", proxyHandler(userServiceURL+"/api/users/profile"))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("API Gateway starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Error during proxyHandler.ReadAll: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read request body"})
			return
		}

		client := resty.New()
		request := client.R()
		request.SetBody(body)
		request.Method = c.Request.Method
		request.SetHeaderMultiValues(c.Request.Header)
		fmt.Printf("%+v\n", request)

		response, err := request.Execute(c.Request.Method, targetURL)
		if err != nil {
			log.Printf("Error during proxyHandler.NewRequest: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to proxy request"})
			return
		}

		for name, values := range response.Header() {
			for _, value := range values {
				c.Header(name, value)
			}
		}
		c.Status(response.StatusCode())
		if _, err = c.Writer.Write(response.Body()); err != nil {
			log.Printf("Error during proxyHandler.Write: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write response"})
		}
	}
}
