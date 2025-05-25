package repositories

import (
	"social-network/statistics-service/models"
	"time"
)

type IRepository interface {
	SaveEvent(event *models.Event) error
	GetPostStats(postID uint) (*models.PostStat, error)
	GetPostTimeline(postID uint, eventType models.EventType, startDate, endDate time.Time) ([]models.TimelinePoint, error)
	GetTopPosts(eventType models.EventType, limit int) ([]models.TopItem, error)
	GetTopUsers(eventType models.EventType, limit int) ([]models.TopItem, error)
}
