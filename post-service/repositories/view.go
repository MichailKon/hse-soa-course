package repositories

import (
	"context"
	"gorm.io/gorm"
	"social-network/post-service/models"
)

type ViewRepository struct {
	db *gorm.DB
}

func NewViewRepository(db *gorm.DB) *ViewRepository {
	return &ViewRepository{db}
}

func (r *ViewRepository) RecordView(ctx context.Context, view *models.View) error {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.View{}).
		Where("post_id = ?", view.PostID).
		Where("viewer_id = ?", view.ViewerID).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return r.db.WithContext(ctx).Create(view).Error
	}
	return nil
}
