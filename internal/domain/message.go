package domain

import (
	"context"
	"time"
)

type Message struct {
	Text string    `json:"text"`
	At   time.Time `json:"at"`
	By   string    `json:"by"`
	Chat int       `json:"chat"`
}

type MessageStore interface {
	SaveMessage(ctx context.Context, msg Message) error
	GetRecentMessages(ctx context.Context, chatID int, limit int) ([]Message, error)
}
