package models

import (
	"time"
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
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatorID   string    `json:"creator_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsPrivate   bool      `json:"is_private"`
	Tags        []string  `json:"tags"`
}

type ListPostsResponse struct {
	Posts      []Post `json:"posts"`
	TotalCount int32  `json:"total_count"`
	TotalPages int32  `json:"total_pages"`
	Page       int32  `json:"page"`
	PageSize   int32  `json:"page_size"`
}
