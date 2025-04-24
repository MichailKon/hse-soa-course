package models

import "gorm.io/gorm"

type Post struct {
	gorm.Model
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description"`
	CreatorID   string `json:"creator_id" gorm:"index;not null"`
	IsPrivate   bool   `json:"is_private" gorm:"default:false"`
	Tags        []Tag  `json:"tags" gorm:"many2many:post_tags;"`
}

type Tag struct {
	ID    uint   `gorm:"primaryKey"`
	Name  string `gorm:"not null;uniqueIndex"`
	Posts []Post `gorm:"many2many:post_tags;"`
}

type PostTag struct {
	PostID uint `gorm:"not null"`
	TagID  uint `gorm:"not null"`
}
