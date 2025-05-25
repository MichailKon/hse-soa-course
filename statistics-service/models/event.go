package models

import "time"

type EventType string

const (
	EventTypeView    EventType = "view"
	EventTypeLike    EventType = "like"
	EventTypeComment EventType = "comment"
)

type Event struct {
	EventType EventType `ch:"event_type"`
	UserID    string    `ch:"user_id"`
	PostID    uint      `ch:"post_id"`
	Timestamp time.Time `ch:"timestamp"`
	Date      time.Time `ch:"date"`
}

type PostStat struct {
	PostID        uint   `ch:"post_id"`
	ViewsCount    uint64 `ch:"views_count"`
	LikesCount    uint64 `ch:"likes_count"`
	CommentsCount uint64 `ch:"comments_count"`
}

type TimelinePoint struct {
	Date  time.Time `ch:"date"`
	Count uint64    `ch:"count"`
}

type TopItem struct {
	ID    uint64 `ch:"id"`
	Title string `ch:"title"`
	Count uint64 `ch:"count"`
}
