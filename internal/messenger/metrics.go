package messenger

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	messagesAccepted  atomic.Int64
	messagesDropped   atomic.Int64
	messagesSampled   atomic.Int64
	chatsCleanedUp    atomic.Int64
	readersCleanedUp  atomic.Int64
	activeConnections atomic.Int64
}

func (m *messenger) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]int64{
		"messages_accepted":  m.metrics.messagesAccepted.Load(),
		"messages_dropped":   m.metrics.messagesDropped.Load(),
		"messages_sampled":   m.metrics.messagesSampled.Load(),
		"chats_cleaned_up":   m.metrics.chatsCleanedUp.Load(),
		"readers_cleaned_up": m.metrics.readersCleanedUp.Load(),
		"active_connections": m.metrics.activeConnections.Load(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
