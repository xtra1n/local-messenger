package messenger

import (
	"context"
	"fmt"
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

func fnv32(s string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int(h.Sum32())
}

func (m *messenger) HandleWS(w http.ResponseWriter, r *http.Request) {
	chatID, user, err := parseWSParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	conn, err := upgradeToWebSocket(w, r)
	if err != nil {
		m.log.Error("websocket upgrade error: ", err)
		return
	}
	defer conn.Close()

	m.metrics.activeConnections.Add(1)
	defer m.metrics.activeConnections.Add(-1)

	deviceID := int(time.Now().UnixNano())
	ch := m.subscribe(chatID, deviceID, user)
	defer m.unsubscribe(chatID, deviceID, user)

	if err := m.sendHistory(conn, chatID, user); err != nil {
		m.log.Error("failed to send history chat=", chatID, " user=", user, " err=", err)
	}

	go m.consumeClienMessages(conn, chatID, deviceID)

	m.streamFromChannel(r.Context(), conn, ch)
}

func parseWSParams(r *http.Request) (int, string, error) {
	chatStr := r.URL.Query().Get("chat")
	user := r.URL.Query().Get("user")

	if chatStr == "" || user == "" {
		return 0, "", fmt.Errorf("chat and user query params requires")
	}

	chatID, err := strconv.Atoi(chatStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid chat id")
	}

	return chatID, user, nil
}

func upgradeToWebSocket(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

func (m *messenger) subscribe(chatID, deviceID int, user string) <-chan Message {
	ch := m.listeners.Get(chatID, deviceID)
	m.log.Info("websocket connected chat=", chatID, " user=", user)
	return ch
}

func (m *messenger) unsubscribe(chatID, deviceID int, user string) {
	m.listeners.Remove(chatID, deviceID)
	m.log.Info("websocket removed chat=", chatID, " user=", user)
}

func (m *messenger) sendHistory(conn *websocket.Conn, chatID int, user string) error {
	history := m.getHistory(chatID)

	for _, msg := range history {
		if err := conn.WriteJSON(msg); err != nil {
			m.log.Error("websocket history write error chat=", chatID, " user=", user, " err=", err)
			return err
		}
	}

	return nil
}

func (m *messenger) consumeClienMessages(conn *websocket.Conn, chatID, deviceID int) {
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			m.log.Info("websocket read closed chat=", chatID, " device=", deviceID, " err=", err)
			return
		}
	}
}

func (m *messenger) streamFromChannel(ctx context.Context, conn *websocket.Conn, ch <-chan Message) {
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteJSON(msg); err != nil {
				if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					return
				}
				m.log.Error("websocket write error: ", err)
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
