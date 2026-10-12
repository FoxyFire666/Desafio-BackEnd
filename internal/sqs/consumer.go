package sqs

import (
	"context"
	"fmt"
	"time"
)

type MessageEnvelope struct {
	MessageID  string    `json:"messageId"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurredAt"`
	Data       map[string]any `json:"data"`
}

type Consumer struct {
	QueueURL string
}

func NewConsumer(queueURL string) *Consumer {
	return &Consumer{QueueURL: queueURL}
}

func (c *Consumer) Process(ctx context.Context, message MessageEnvelope) error {
	if c.QueueURL == "" {
		return nil
	}
	if message.MessageID == "" {
		return fmt.Errorf("message id is required")
	}
	return nil
}
