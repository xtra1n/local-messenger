package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/xtra1n/local-messenger/pkg/logger"
)

type messenger struct {
	input     chan Message
	log       *logger.Logger
	metrics   *Metrics
	listeners *listenerMap

	store MessageStore
}

func New(log *logger.Logger, store MessageStore) Messenger {
	return &messenger{
		input:     make(chan Message, 1000),
		log:       log,
		metrics:   &Metrics{},
		listeners: newListenersMap(),
		store:     store,
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
	ctx := r.Context()

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
		_, _ = w.Write([]byte("system overloaded, try again later"))
		return
	}

	if m.store != nil {
		if err := m.store.SaveMessage(ctx, msg); err != nil {
			m.log.Error("failed to save message to store: ", err)
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("message sent"))
}

func (m *messenger) Subscribe(chatID int, deviceID int) <-chan Message {
	return m.listeners.Get(chatID, deviceID)
}

func (m *messenger) getHistory(ctx context.Context, chatID int) []Message {
	if m.store == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	msgs, err := m.store.GetRecentMessages(ctx, chatID, 100)
	if err != nil {
		m.log.Error("failed to load history from store chat=", chatID, " err=", err)
		return nil
	}

	return msgs
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
