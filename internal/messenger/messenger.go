package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/xtra1n/local-messenger/internal/domain"
	"github.com/xtra1n/local-messenger/pkg/logger"
)

type messenger struct {
	input     chan domain.Message
	log       *logger.Logger
	metrics   *Metrics
	listeners *listenerMap
	store     domain.MessageStore
}

func New(log *logger.Logger, store domain.MessageStore) domain.Messenger {
	return &messenger{
		input:     make(chan domain.Message, 1000),
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
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte("method not allowed"))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 10<<10)

	msg, err := m.decodeMessage(r)
	if err != nil {
		if strings.Contains(err.Error(), "http: request body too large") {
			http.Error(w, "message too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := validateMessage(&msg); err != nil {
		http.Error(w, "validation error: "+err.Error(), http.StatusBadRequest)
		return
	}

	msg.At = time.Now()

	if !m.enqueueMessage(msg) {
		m.metrics.MessagesDropped.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("system overloaded, try again later"))
		return
	}

	if m.store != nil {
		go func() {
			saveCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := m.store.SaveMessage(saveCtx, msg); err != nil {
				m.log.Error("failed to save message to store", "chat", msg.Chat, "err", err)
			}
		}()
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("message sent"))
}

func (m *messenger) Subscribe(chatID int, deviceID int) <-chan domain.Message {
	return m.listeners.Get(chatID, deviceID)
}

func (m *messenger) getHistory(ctx context.Context, chatID int) []domain.Message {
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

func (m *messenger) decodeMessage(r *http.Request) (domain.Message, error) {
	var msg domain.Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		return domain.Message{}, err
	}

	return msg, nil
}

func (m *messenger) enqueueMessage(msg domain.Message) bool {
	select {
	case m.input <- msg:
		m.metrics.MessagesAccepted.Add(1)
		return true
	case <-time.After(5 * time.Second):
		return false
	}
}
