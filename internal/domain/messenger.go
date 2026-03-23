package domain

import (
	"context"
	"net/http"
)

type Messenger interface {
	Run(ctx context.Context) error
	AddMessage(w http.ResponseWriter, r *http.Request)
	MetricsHandler(w http.ResponseWriter, r *http.Request)
	Subscribe(chatID int, deviceID int) <-chan Message
	HandleWS(w http.ResponseWriter, r *http.Request)
}
