package models

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	Username    string     `json:"username" gorm:"uniqueIndex;not null"`
	Password    string     `json:"-" gorm:"not null"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	Email       string     `json:"email" gorm:"uniqueIndex;not null"`
	BirthDate   *time.Time `json:"birth_date"`
	PhoneNumber string     `json:"phone_number"`
}
