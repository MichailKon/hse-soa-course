package main

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"api-gateway/handlers"
	"api-gateway/middleware"
	"api-gateway/proto"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	if userServiceURL == "" {
		userServiceURL = "http://user-service:8081"
	}
	postServiceURL := os.Getenv("POST_SERVICE_URL")
	if postServiceURL == "" {
		postServiceURL = "post-service:50051"
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "JWT_SECRET"
	}
	conn, err := grpc.NewClient(
		postServiceURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to post service: %v", err)
	}
	defer conn.Close()
	postClient := proto.NewPostServiceClient(conn)
	postHandler := handlers.NewPostHandler(postClient)
	jwtKey := []byte(jwtSecret)
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	api := router.Group("/api")
	api.POST("/auth/register", proxyHandler(userServiceURL+"/api/auth/register"))
	api.POST("/auth/login", proxyHandler(userServiceURL+"/api/auth/login"))
	api.GET("/users/profile", proxyWithAuthHandler(userServiceURL+"/api/users/profile", jwtKey))
	api.PUT("/users/profile", proxyWithAuthHandler(userServiceURL+"/api/users/profile", jwtKey))
	posts := api.Group("/posts")
	posts.Use(middleware.AuthMiddleware(jwtKey))
	{
		posts.POST("", postHandler.CreatePost)
		posts.GET("/:id", postHandler.GetPost)
		posts.PUT("/:id", postHandler.UpdatePost)
		posts.DELETE("/:id", postHandler.DeletePost)
		posts.GET("", postHandler.ListPosts)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("API Gateway listening on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

func proxyHandler(targetURL string) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		targetPath := target.Path
		if targetPath == "" {
			targetPath = req.URL.Path
		}
		req.URL.Path = targetPath
	}
	return func(context *gin.Context) {
		proxy.ServeHTTP(context.Writer, context.Request)
	}
}

func proxyWithAuthHandler(targetURL string, jwtKey []byte) gin.HandlerFunc {
	target, err := url.Parse(targetURL)
	if err != nil {
		log.Fatalf("Failed to parse URL: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be 'Bearer {token}'"})
			c.Abort()
			return
		}
		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		originalPath := c.Request.URL.Path
		if strings.HasPrefix(originalPath, "/api") {
			c.Request.URL.Path = strings.TrimPrefix(originalPath, "/api")
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
