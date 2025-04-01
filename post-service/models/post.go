package models

import (
	"time"
)

type Post struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null"`
	Description string    `json:"description"`
	CreatorID   string    `json:"creator_id" gorm:"index;not null"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsPrivate   bool      `json:"is_private" gorm:"default:false"`
	Tags        []Tag     `json:"tags" gorm:"many2many:post_tags;"`
}

type Tag struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"not null;uniqueIndex"`
	Posts []Post `gorm:"many2many:post_tags;"`
}

type PostTag struct {
	PostID string `gorm:"not null"`
	TagID  uint   `gorm:"not null"`
}
