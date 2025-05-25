package kafka

import (
	"context"
	"encoding/json"
	"log"
	commonkafka "social-network/common/kafka"
	"social-network/statistics-service/models"
	"social-network/statistics-service/repositories"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	readers          map[string]*kafka.Reader
	chRepository     *repositories.ClickHouseRepository
	bootstrapServers string
}

func NewConsumer(bootstrapServers string, chRepository *repositories.ClickHouseRepository) *Consumer {
	return &Consumer{
		readers:          make(map[string]*kafka.Reader),
		chRepository:     chRepository,
		bootstrapServers: bootstrapServers,
	}
}

func (c *Consumer) getReader(topic string) *kafka.Reader {
	if reader, exists := c.readers[topic]; exists {
		return reader
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{c.bootstrapServers},
		Topic:   topic,
		GroupID: "statistics-service",
	})

	c.readers[topic] = reader
	return reader
}

func (c *Consumer) ConsumeEvents(ctx context.Context) {
	topics := []string{"post_views", "post_likes", "post_comments"}

	for _, topic := range topics {
		go c.consumeTopic(ctx, topic)
	}
}

func (c *Consumer) consumeTopic(ctx context.Context, topic string) {
	reader := c.getReader(topic)
	defer reader.Close()

	log.Printf("Starting to consume from topic: %s", topic)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			msg, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("Error reading message from Kafka topic %s: %v", topic, err)
				continue
			}

			var event commonkafka.Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("Error unmarshalling Kafka message: %v", err)
				continue
			}

			var eventType models.EventType
			switch topic {
			case "post_views":
				eventType = models.EventTypeView
			case "post_likes":
				eventType = models.EventTypeLike
			case "post_comments":
				eventType = models.EventTypeComment
			default:
				log.Printf("Unknown topic: %s", topic)
				continue
			}

			chEvent := &models.Event{
				EventType: eventType,
				UserID:    event.UserID,
				PostID:    event.EntityID,
				Timestamp: time.Unix(0, event.Timestamp*int64(time.Millisecond)),
				Date:      time.Unix(0, event.Timestamp*int64(time.Millisecond)).Truncate(24 * time.Hour),
			}

			if err = c.chRepository.SaveEvent(chEvent); err != nil {
				log.Printf("Error saving event to ClickHouse: %v", err)
				continue
			}
		}
	}
}

func (c *Consumer) Close() {
	for _, reader := range c.readers {
		reader.Close()
	}
}
