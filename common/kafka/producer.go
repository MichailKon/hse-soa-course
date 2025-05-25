package kafka

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"os"
	"time"
)

type Producer struct {
	writers map[string]*kafka.Writer
}

type Event struct {
	EventType string      `json:"event_type"`
	UserID    string      `json:"user_id"`
	EntityID  uint        `json:"entity_id"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

func NewProducer() *Producer {
	bootstrapServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if bootstrapServers == "" {
		bootstrapServers = "kafka:9092"
	}
	return &Producer{
		writers: make(map[string]*kafka.Writer),
	}
}

func (p *Producer) getWriter(topic string) *kafka.Writer {
	if writer, exists := p.writers[topic]; exists {
		return writer
	}

	bootstrapServers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if bootstrapServers == "" {
		bootstrapServers = "kafka:9092"
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(bootstrapServers),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	p.writers[topic] = writer
	return writer
}

func (p *Producer) SendEvent(topic string, event *Event) error {
	value, err := json.Marshal(event)
	if err != nil {
		return err
	}

	writer := p.getWriter(topic)
	return writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: value,
		},
	)
}

func (p *Producer) Close() error {
	var lastErr error
	for _, writer := range p.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func NewEvent(eventType string, userID string, entityID uint, data interface{}) *Event {
	return &Event{
		EventType: eventType,
		UserID:    userID,
		EntityID:  entityID,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
}
