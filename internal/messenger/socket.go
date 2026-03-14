package messenger

import (
	"hash/fnv"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (m *messenger) HandleWS(w http.ResponseWriter, r *http.Request) {
	chatStr := r.URL.Query().Get("chat")
	user := r.URL.Query().Get("user")

	if chatStr == "" || user == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("chat and user query params required"))
		return
	}

	chatID, err := strconv.Atoi(chatStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid chat id"))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		m.log.Error("websocket upgrade error: ", err)
		return
	}
	defer conn.Close()

	m.metrics.activeConnections.Add(1)
	defer m.metrics.activeConnections.Add(-1)

	deviceID := fnv32(user)

	ch := m.listeners.Get(chatID, deviceID)
	m.log.Info("websocket connected chat=", chatID, " user=", user)

	history := m.getHistory(chatID)
	for _, msg := range history {
		if err := conn.WriteJSON(msg); err != nil {
			m.log.Error("websocket history write error chat=", chatID, " user=", user, " err=", err)
			return
		}
	}

	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				m.log.Info("websocket read error (closing) chat=", chatID, " device=", deviceID, " err=", err)
				return
			}
		}
	}()

	for {
		select {
		case msg := <-ch:
			if err := conn.WriteJSON(msg); err != nil {
				m.log.Error("websocket write error: ", err)
				return
			}
		case <-r.Context().Done():
			return
		case <-time.After(30 * time.Minute):
			return
		}
	}

}

func fnv32(s string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int(h.Sum32())
}

func (m *messenger) getHistory(chatID int) []Message {
	m.historyMu.RLock()
	defer m.historyMu.RUnlock()

	msgs := m.history[chatID]
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out
}
