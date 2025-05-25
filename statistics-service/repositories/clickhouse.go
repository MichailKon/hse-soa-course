package repositories

import (
	"context"
	"fmt"
	"log"
	"social-network/statistics-service/models"
	"strconv"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHouseRepository struct {
	conn driver.Conn
}

func NewClickHouseRepository(host, port, database, username, password string) (*ClickHouseRepository, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     time.Second * 10,
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
	})

	if err != nil {
		log.Printf("Fail in connect")
		return nil, err
	}

	repo := &ClickHouseRepository{conn: conn}
	if err := repo.createTables(); err != nil {
		log.Printf("Fail in create tables")
		return nil, err
	}

	return repo, nil
}

func (r *ClickHouseRepository) createTables() error {
	return r.conn.Exec(context.Background(),
		`CREATE TABLE IF NOT EXISTS events (
			event_type String,
			user_id String,
			post_id UInt64,
			timestamp DateTime,
			date Date
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(date)
		ORDER BY (post_id, date, event_type)`)
}

func (r *ClickHouseRepository) SaveEvent(event *models.Event) error {
	return r.conn.Exec(context.Background(),
		"INSERT INTO events (event_type, user_id, post_id, timestamp, date) VALUES (?, ?, ?, ?, ?)",
		event.EventType, event.UserID, event.PostID, event.Timestamp, event.Date)
}

func (r *ClickHouseRepository) GetPostStats(postID uint) (*models.PostStat, error) {
	ctx := context.Background()

	var viewsCount, likesCount, commentsCount uint64

	err := r.conn.QueryRow(ctx,
		"SELECT COUNT(*) FROM events WHERE post_id = ? AND event_type = ?",
		postID, models.EventTypeView).Scan(&viewsCount)
	if err != nil {
		return nil, err
	}

	err = r.conn.QueryRow(ctx,
		"SELECT COUNT(*) FROM events WHERE post_id = ? AND event_type = ?",
		postID, models.EventTypeLike).Scan(&likesCount)
	if err != nil {
		return nil, err
	}

	err = r.conn.QueryRow(ctx,
		"SELECT COUNT(*) FROM events WHERE post_id = ? AND event_type = ?",
		postID, models.EventTypeComment).Scan(&commentsCount)
	if err != nil {
		return nil, err
	}

	return &models.PostStat{
		PostID:        postID,
		ViewsCount:    viewsCount,
		LikesCount:    likesCount,
		CommentsCount: commentsCount,
	}, nil
}

func (r *ClickHouseRepository) GetPostTimeline(postID uint, eventType models.EventType, startDate, endDate time.Time) ([]models.TimelinePoint, error) {
	rows, err := r.conn.Query(context.Background(),
		`SELECT date, COUNT(*) as count 
		FROM events 
		WHERE post_id = ? AND event_type = ? AND date BETWEEN ? AND ? 
		GROUP BY date 
		ORDER BY date`,
		postID, eventType, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TimelinePoint
	for rows.Next() {
		var point models.TimelinePoint
		if err := rows.Scan(&point.Date, &point.Count); err != nil {
			return nil, err
		}
		results = append(results, point)
	}

	return results, rows.Err()
}

func (r *ClickHouseRepository) GetTopPosts(eventType models.EventType, limit int) ([]models.TopItem, error) {
	rows, err := r.conn.Query(context.Background(),
		`SELECT post_id as id, COUNT(*) as count 
		FROM events 
		WHERE event_type = ? 
		GROUP BY post_id 
		ORDER BY count DESC 
		LIMIT ?`,
		eventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TopItem
	for rows.Next() {
		var item models.TopItem
		if err := rows.Scan(&item.ID, &item.Count); err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	return results, rows.Err()
}

func (r *ClickHouseRepository) GetTopUsers(eventType models.EventType, limit int) ([]models.TopItem, error) {
	rows, err := r.conn.Query(context.Background(),
		`SELECT user_id as id, COUNT(*) as count 
		FROM events 
		WHERE event_type = ? 
		GROUP BY user_id 
		ORDER BY count DESC 
		LIMIT ?`,
		eventType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.TopItem
	for rows.Next() {
		var userID string
		var item models.TopItem
		if err := rows.Scan(&userID, &item.Count); err != nil {
			return nil, err
		}
		if id, err := strconv.ParseInt(userID, 10, 64); err == nil {
			item.ID = uint64(id)
			results = append(results, item)
		} else {
			return nil, err
		}
	}

	return results, rows.Err()
}
