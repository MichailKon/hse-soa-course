package contracts

import (
	"social-network/user-service/models"
	"time"
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	// max=72 because of https://pkg.go.dev/golang.org/x/crypto/bcrypt@v0.35.0#GenerateFromPassword
	Password    string     `json:"password" binding:"required,min=6,max=72"`
	FirstName   string     `json:"first_name" binding:"omitempty,max=50"`
	LastName    string     `json:"last_name" binding:"omitempty,max=50"`
	Email       string     `json:"email" binding:"required,email"`
	BirthDate   *time.Time `json:"birth_date" binding:"omitempty" time_format:"1970-01-01"`
	PhoneNumber string     `json:"phone_number" binding:"omitempty"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	FirstName   string     `json:"first_name" binding:"omitempty,max=50"`
	LastName    string     `json:"last_name" binding:"omitempty,max=50"`
	Email       string     `json:"email" binding:"omitempty,email"`
	BirthDate   *time.Time `json:"birth_date" binding:"omitempty"`
	PhoneNumber string     `json:"phone_number" binding:"omitempty"`
}

type AuthResponse struct {
	JwtToken string      `json:"jwt_token"`
	User     models.User `json:"user"`
}

type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
}
