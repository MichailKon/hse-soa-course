package repositories

import (
	"context"
	"gorm.io/gorm"
	"social-network/post-service/models"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db}
}

func (r *CommentRepository) CreateComment(ctx context.Context, comment *models.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *CommentRepository) GetCommentsForPostByID(
	ctx context.Context, postID, page, pageSize int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var count int64
	if err := r.db.
		WithContext(ctx).
		Model(&models.Comment{}).
		Where("post_id = ?", postID).
		Count(&count).Error; err != nil {
		return nil, 0, err
	}
	if err := r.db.
		WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, count, nil
}
