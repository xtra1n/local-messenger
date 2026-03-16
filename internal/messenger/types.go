package messenger

import (
	"context"
	"net/http"
	"time"
)

type Message struct {
	Text string    `json:"text"`
	At   time.Time `json:"at"`
	By   string    `json:"by"`
	Chat int       `json:"chat"`
}

type Messenger interface {
	Run(ctx context.Context) error
	AddMessage(w http.ResponseWriter, r *http.Request)
	MetricsHandler(w http.ResponseWriter, r *http.Request)
	Subscribe(chatID int, deviceID int) <-chan Message
	HandleWS(w http.ResponseWriter, r *http.Request)
}

type MessageStore interface {
	SaveMessage(ctx context.Context, msg Message) error
	GetRecentMessages(ctx context.Context, chatID int, limit int) ([]Message, error)
}