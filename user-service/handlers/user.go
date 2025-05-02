package handlers

import (
	"fmt"
	"log"
	"net/http"
	"social-network/common/kafka"
	"social-network/user-service/contracts"
	"social-network/user-service/models"
	"social-network/user-service/repositories"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	UserRepo      *repositories.UserRepository
	KafkaProducer *kafka.Producer
	JwtKey        []byte
}

func NewUserHandler(userRepo *repositories.UserRepository, jwtKey string, producer *kafka.Producer) *UserHandler {
	return &UserHandler{
		UserRepo:      userRepo,
		KafkaProducer: producer,
		JwtKey:        []byte(jwtKey),
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var registerRequest contracts.RegisterRequest
	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingUser, err := h.UserRepo.FindByUsername(registerRequest.Username)
	if err != nil {
		log.Printf("Error during Register.FindByUsername: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking username"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	existingUser, err = h.UserRepo.FindByEmail(registerRequest.Email)
	if err != nil {
		log.Printf("Error during Register.FindByEmail: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking email"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	user := models.User{
		Username:    registerRequest.Username,
		Email:       registerRequest.Email,
		Password:    registerRequest.Password,
		FirstName:   registerRequest.FirstName,
		LastName:    registerRequest.LastName,
		BirthDate:   registerRequest.BirthDate,
		PhoneNumber: registerRequest.PhoneNumber,
	}

	err = h.UserRepo.CreateUser(&user)
	if err != nil {
		log.Printf("Error during Register.CreateUser: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	fmt.Println(user.ID)

	jwtToken, err := h.generateJwtToken(user.ID, user.Username)
	if err != nil {
		log.Printf("Error during Register.generateJwtToken: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate jwtToken"})
		return
	}

	event := kafka.NewEvent("user_events", fmt.Sprint(user.ID), user.ID, map[string]any{
		"username":   user.Username,
		"email":      user.Email,
		"created_at": user.CreatedAt,
	})
	if h.KafkaProducer != nil {
		if err = h.KafkaProducer.SendEvent("user_events", event); err != nil {
			log.Printf("Error during Register.kafka.SendEvent: %v", err)
		}
	}

	c.JSON(http.StatusCreated, contracts.AuthResponse{
		JwtToken: jwtToken,
		User:     user,
	})
}

func (h *UserHandler) Login(c *gin.Context) {
	var loginRequest contracts.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.UserRepo.FindByUsername(loginRequest.Username)
	if err != nil {
		log.Printf("Error during Login.FindByUsername: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginRequest.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	jwtToken, err := h.generateJwtToken(user.ID, user.Username)
	if err != nil {
		log.Printf("Error during Login.generateJwtToken: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate jwtToken"})
		return
	}

	c.JSON(http.StatusOK, contracts.AuthResponse{
		JwtToken: jwtToken,
		User:     *user,
	})
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	fmt.Println(userID, exists)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.UserRepo.FindByID(userID.(uint))
	if err != nil {
		log.Printf("Error during GetProfile.FindByID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var updateRequest contracts.UpdateProfileRequest
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.UserRepo.FindByID(userID.(uint))
	if err != nil {
		log.Printf("Error during UpdateProfile.FindByID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if updateRequest.FirstName != "" {
		user.FirstName = updateRequest.FirstName
	}
	if updateRequest.LastName != "" {
		user.LastName = updateRequest.LastName
	}
	if updateRequest.Email != "" && updateRequest.Email != user.Email {
		existingUser, err := h.UserRepo.FindByEmail(updateRequest.Email)
		if err != nil {
			log.Printf("Error during UpdateProfile.FindByEmail: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking email"})
			return
		}
		if existingUser != nil && existingUser.ID != user.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
			return
		}
		user.Email = updateRequest.Email
	}
	if updateRequest.BirthDate != nil {
		user.BirthDate = updateRequest.BirthDate
	}
	if updateRequest.PhoneNumber != "" {
		user.PhoneNumber = updateRequest.PhoneNumber
	}

	if err = h.UserRepo.UpdateUser(user); err != nil {
		log.Printf("Error during UpdateProfile.UpdateUser: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) generateJwtToken(userID uint, username string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      expirationTime.Unix(),
	})

	tokenString, err := token.SignedString(h.JwtKey)

	return tokenString, err
}
