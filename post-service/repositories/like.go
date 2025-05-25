package repositories

import (
	"context"
	"gorm.io/gorm"
	"social-network/post-service/models"
)

type LikeRepository struct {
	db *gorm.DB
}

func NewLikeRepository(db *gorm.DB) *LikeRepository {
	return &LikeRepository{db}
}

func (r *LikeRepository) LikesCountByPostID(ctx context.Context, postID uint64) (int64, error) {
	var count int64
	if err := r.db.
		WithContext(ctx).
		Model(&models.Like{}).
		Where("post_id = ?", postID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// ToggleLike return true if post is liked now (value doesn't mean anything in case of error)
func (r *LikeRepository) ToggleLike(ctx context.Context, like *models.Like) (bool, error) {
	var post models.Post
	if err := r.db.
		WithContext(ctx).
		Model(&models.Post{}).
		First(&post, like.PostID).Error; err != nil {
		return false, err
	}

	var existingLike models.Like
	result := r.db.
		WithContext(ctx).
		Where("post_id = ?", like.PostID).
		Where("liker_id = ?", like.LikerID).
		First(&existingLike)
	if result.Error == nil {
		return false, r.db.Delete(&existingLike).Error
	}
	return true, r.db.WithContext(ctx).Create(like).Error
}
