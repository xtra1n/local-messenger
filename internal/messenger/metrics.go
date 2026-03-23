package messenger

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
)

type Metrics struct {
	MessagesAccepted  atomic.Int64
	MessagesDropped   atomic.Int64
	MessagesSampled   atomic.Int64
	ChatsCleanedUp    atomic.Int64
	ReadersCleanedUp  atomic.Int64
	ActiveConnections atomic.Int64
}

func (m *messenger) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]int64{
		"messages_accepted":  m.metrics.MessagesAccepted.Load(),
		"messages_dropped":   m.metrics.MessagesDropped.Load(),
		"messages_sampled":   m.metrics.MessagesSampled.Load(),
		"chats_cleaned_up":   m.metrics.ChatsCleanedUp.Load(),
		"readers_cleaned_up": m.metrics.ReadersCleanedUp.Load(),
		"active_connections": m.metrics.ActiveConnections.Load(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
