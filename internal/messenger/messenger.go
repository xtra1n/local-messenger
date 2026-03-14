package messenger

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/xtra1n/local-messenger/pkg/logger"
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

	return ctx.Err()
}

func (m *messenger) distributor(ctx context.Context) {
	m.log.Info("distributor worker started")

	for {
		select {
		case <-ctx.Done():
			m.log.Info("distributor worker stopping")
			return
		case msg := <-m.input:
			listeneres := m.listeners.GetChatListeners(msg.Chat)
			if len(listeneres) == 0 {
				m.log.Debug("no listeners for chat ", msg.Chat)
				continue
			}

			for deviceID, ch := range listeneres {
				select {
				case ch <- msg:
					m.metrics.messagesSampled.Add(1)
				default:
					m.log.Debug("listener channel full, chat=", msg.Chat, " device=", deviceID)
				}
			}
		}
	}
}

func (m *messenger) AddMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var msg Message
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid JSON"))
		return
	}

	msg.At = time.Now()

	select {
	case m.input <- msg:
		m.metrics.messagesAccepted.Add(1)

		m.historyMu.Lock()
		msgs := m.history[msg.Chat]
		msgs = append(msgs, msg)

		if len(msgs) > 100 {
			msgs = msgs[len(msgs) - 100:]
		}

		m.history[msg.Chat] = msgs
		m.historyMu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("message sent"))
	case <-time.After(5 * time.Second):
		m.metrics.messagesDropped.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("system overloaded, try again later"))
	}
}

func (m *messenger) Subscribe(chatID int, deviceID int) <-chan Message {
	return m.listeners.Get(chatID, deviceID)
}
