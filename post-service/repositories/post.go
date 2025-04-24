package repositories

import (
	"errors"
	"gorm.io/gorm"
	"post-service/models"
)

type PostRepository struct {
	db *gorm.DB
}

func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

func (r *PostRepository) CreatePost(post *models.Post) error {
	tagNames := make([]string, len(post.Tags))
	for i, tag := range post.Tags {
		tagNames[i] = tag.Name
	}
	post.Tags = nil
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(post).Error; err != nil {
			return err
		}
		for _, tagName := range tagNames {
			var tag models.Tag
			if tx.Where("name = ?", tagName).First(&tag).Error != nil {
				tag.Name = tagName
				if err := tx.Create(&tag).Error; err != nil {
					return err
				}
			}
			if err := tx.Create(&models.PostTag{
				PostID: post.ID,
				TagID:  tag.ID,
			}).Error; err != nil {
				return err
			}
		}
		return tx.Preload("Tags").First(post, "id = ?", post.ID).Error
	})
}

func (r *PostRepository) GetPostByID(id uint64) (*models.Post, error) {
	var post models.Post

	if err := r.db.Preload("Tags").First(&post, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &post, nil
}

func (r *PostRepository) UpdatePost(post *models.Post, tagNames []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(post).Error; err != nil {
			return err
		}
		if err := tx.Model(post).Association("Tags").Clear(); err != nil {
			return err
		}
		var tags []models.Tag
		for _, tagName := range tagNames {
			var tag models.Tag
			result := tx.Where("name = ?", tagName).First(&tag)
			if result.Error != nil {
				tag.Name = tagName
				if err := r.db.Create(&tag).Error; err != nil {
					return err
				}
			}
			tags = append(tags, tag)
		}
		if err := tx.Model(post).Association("Tags").Replace(tags); err != nil {
			return err
		}
		return nil
	})
}

func (r *PostRepository) DeletePost(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var post models.Post
		if err := tx.First(&post, "id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Model(&post).Association("Tags").Clear(); err != nil {
			return err
		}
		if err := tx.Delete(&post).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *PostRepository) ListPosts(
	page, pageSize int,
	creatorID string,
	tagNames []string,
	includePrivate bool,
	requesterID string) ([]models.Post, int64, error) {
	query := r.db.Model(&models.Post{})
	var count int64
	if creatorID != "" {
		query = query.Where("creator_id = ?", creatorID)
	}
	if !includePrivate {
		query = query.Where("is_private = ? OR creator_id = ?", false, requesterID)
	}
	if len(tagNames) > 0 {
		query = query.Joins("JOIN post_tags ON post_tags.post_id = posts.id").
			Joins("JOIN tags ON tags.id = post_tags.tag_id").
			Where("tags.name IN ?", tagNames).
			Group("posts.id")
	}
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	var posts []models.Post
	if err := query.Preload("Tags").Offset((page - 1) * pageSize).Limit(pageSize).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, count, nil
}
