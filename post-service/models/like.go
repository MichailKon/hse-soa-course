package models

import "gorm.io/gorm"

type Like struct {
	gorm.Model
	PostID  uint   `json:"post_id" gorm:"index,not null"`
	LikerID uint64 `json:"liker_id" gorm:"not null"`
	Post    Post   `json:"post" gorm:"foreignKey:PostID"`
}
