package models

import (
	"gorm.io/gorm"
)

type CreatePostRequest struct {
	Title       string   `json:"title" binding:"required"`
	Description string   `json:"description"`
	IsPrivate   bool     `json:"is_private"`
	Tags        []string `json:"tags"`
}

type UpdatePostRequest struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	IsPrivate   bool     `json:"is_private"`
	Tags        []string `json:"tags"`
}

type Post struct {
	gorm.Model
	Title       string   `json:"title"`
	Description string   `json:"description"`
	CreatorID   string   `json:"creator_id"`
	IsPrivate   bool     `json:"is_private"`
	Tags        []string `json:"tags"`
}

type ListPostsResponse struct {
	Posts      []Post `json:"posts"`
	TotalCount int32  `json:"total_count"`
	TotalPages int32  `json:"total_pages"`
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
}
