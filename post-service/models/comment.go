package models

import "gorm.io/gorm"

type Comment struct {
	gorm.Model
	PostID   uint   `json:"post_id" gorm:"index,not null"`
	AuthorID uint64 `json:"author_id" gorm:"not null"`
	Content  string `json:"content" gorm:"not null"`
	Post     Post   `json:"post" gorm:"foreignKey:PostID"`
}
