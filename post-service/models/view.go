package models

import "gorm.io/gorm"

type View struct {
	gorm.Model
	PostID   uint   `json:"post_id" gorm:"index;not null"`
	ViewerID uint64 `json:"viewer_id" gorm:"not null"`
	Post     Post   `json:"post" gorm:"foreignKey:PostID"`
}
