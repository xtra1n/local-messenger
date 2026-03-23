package messenger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/xtra1n/local-messenger/internal/domain"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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
	defer func() {
		_ = conn.Close()
	}()

	m.metrics.ActiveConnections.Add(1)
	defer m.metrics.ActiveConnections.Add(-1)

	deviceID := int(time.Now().UnixNano())
	ch := m.subscribe(chatID, deviceID, user)
	defer m.unsubscribe(chatID, deviceID, user)

	if err := m.sendHistory(r.Context(), conn, chatID, user); err != nil {
		m.log.Error("failed to send history chat=", chatID, " user=", user, " err=", err)
	}

	go m.consumeClientMessages(conn, chatID, deviceID)

	m.streamFromChannel(r.Context(), conn, ch)
}

func parseWSParams(r *http.Request) (int, string, error) {
	chatStr := r.URL.Query().Get("chat")
	user := r.URL.Query().Get("user")

	if chatStr == "" || user == "" {
		return 0, "", fmt.Errorf("chat and user query params required")
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

func (m *messenger) subscribe(chatID, deviceID int, user string) <-chan domain.Message {
	ch := m.listeners.Get(chatID, deviceID)
	m.log.Info("websocket connected chat=", chatID, " user=", user)
	return ch
}

func (m *messenger) unsubscribe(chatID, deviceID int, user string) {
	m.listeners.Remove(chatID, deviceID)
	m.log.Info("websocket removed chat=", chatID, " user=", user)
}

func (m *messenger) sendHistory(ctx context.Context, conn *websocket.Conn, chatID int, user string) error {
	history := m.getHistory(ctx, chatID)

	for _, msg := range history {
		if err := conn.WriteJSON(msg); err != nil {
			m.log.Error("websocket history write error chat=", chatID, " user=", user, " err=", err)
			return err
		}
	}

	return nil
}

func (m *messenger) consumeClientMessages(conn *websocket.Conn, chatID, deviceID int) {
	m.log.Info("websocket message consumer started", "chat", chatID, "device", deviceID)

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				m.log.Info("websocket client disconnected", "chat", chatID, "device", deviceID)
			} else {
				m.log.Error("websocket read error", "chat", chatID, "device", deviceID, "err", err)
			}
			return
		}

		var msg domain.Message
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			m.log.Warn("failed to decode message from client", "chat", chatID, "device", deviceID, "err", err)
			continue
		}

		if err := validateMessage(&msg); err != nil {
			m.log.Warn("invalid message from client", "chat", chatID, "device", deviceID, "err", err)
			continue
		}

		msg.At = time.Now()
		msg.Chat = chatID

		if !m.enqueueMessage(msg) {
			m.log.Error("failed to enqueue message, system overloaded", "chat", chatID, "device", deviceID)
			errorMsg := domain.Message{
				Text: "SYSTEM_OVERLOAD",
				At:   time.Now(),
				By:   "system",
				Chat: chatID,
			}
			_ = conn.WriteJSON(errorMsg)
			continue
		}
		m.log.Debug("message enqueued successfully", "chat", chatID, "device", deviceID, "text", truncate(msg.Text, 50))
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func validateMessage(msg *domain.Message) error {
	if msg.Text == "" {
		return fmt.Errorf("message text is required")
	}

	if len(msg.Text) > 10000 {
		return fmt.Errorf("message text too long (max 10000 characters)")
	}

	if len(msg.By) == 0 {
		return fmt.Errorf("sender username is required")
	}

	if len(msg.By) > 256 {
		return fmt.Errorf("username too long (max 256 characters)")
	}

	msg.Text = strings.TrimSpace(msg.Text)
	if msg.Text == "" {
		return fmt.Errorf("message text cannot be only whitespace")
	}

	return nil
}

func (m *messenger) streamFromChannel(ctx context.Context, conn *websocket.Conn, ch <-chan domain.Message) {
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
