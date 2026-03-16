package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/xtra1n/local-messenger/pkg/logger"
)

type messenger struct {
	input     chan Message
	log       *logger.Logger
	metrics   *Metrics
	listeners *listenerMap
	historyMu sync.RWMutex
	history   map[int][]Message
}

func New(log *logger.Logger) Messenger {
	return &messenger{
		input:     make(chan Message, 1000),
		log:       log,
		metrics:   &Metrics{},
		listeners: newListnersMap(),
		history:   make(map[int][]Message),
	}
}

func (m *messenger) Run(ctx context.Context) error {
	m.log.Info("messenger workers starting")

	go m.distributor(ctx)

	<-ctx.Done()
	m.log.Info("messenger stopped")

	m.listeners.CloseAll()
	return ctx.Err()
}

func (m *messenger) AddMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	msg, err := m.decodeMessage(r)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	msg.At = time.Now()

	if !m.enqueueMessage(msg) {
		m.metrics.messagesDropped.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("system overloaded, try again later"))
		return
	}

	m.appendToHistory(msg)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("message sent"))
}

func (m *messenger) Subscribe(chatID int, deviceID int) <-chan Message {
	return m.listeners.Get(chatID, deviceID)
}

func (m *messenger) getHistory(chatID int) []Message {
	m.historyMu.RLock()
	defer m.historyMu.RUnlock()

	msgs := m.history[chatID]
	out := make([]Message, len(msgs))
	copy(out, msgs)
	return out
}

func (m *messenger) decodeMessage(r *http.Request) (Message, error) {
	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return Message{}, err
	}

	return msg, nil
}

func (m *messenger) enqueueMessage(msg Message) bool {
	select {
	case m.input <- msg:
		m.metrics.messagesAccepted.Add(1)
		return true
	case <-time.After(5 * time.Second):
		return false
	}
}

func (m *messenger) appendToHistory(msg Message) {
	m.historyMu.Lock()
	defer m.historyMu.Unlock()

	msgs := m.history[msg.Chat]
	msgs = append(msgs, msg)

	if len(msgs) > 100 {
		msgs = msgs[len(msgs)-100:]
	}

	m.history[msg.Chat] = msgs
}
