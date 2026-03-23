package messenger

import (
	"sync"

	"github.com/xtra1n/local-messenger/internal/domain"
)

type listenerMap struct {
	mu   sync.RWMutex
	data map[int]map[int]chan domain.Message
}

func newListenersMap() *listenerMap {
	return &listenerMap{
		data: make(map[int]map[int]chan domain.Message),
	}
}

func (l *listenerMap) Get(chatID int, deviceID int) chan domain.Message {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, ok := l.data[chatID]; !ok {
		l.data[chatID] = make(map[int]chan domain.Message)
	}

	if ch, ok := l.data[chatID][deviceID]; ok {
		return ch
	}

	ch := make(chan domain.Message, 100)
	l.data[chatID][deviceID] = ch

	return ch
}

func (l *listenerMap) GetChatListeners(chatID int) map[int]chan domain.Message {
	l.mu.RLock()
	defer l.mu.RUnlock()

	listeners := make(map[int]chan domain.Message)
	if devices, ok := l.data[chatID]; ok {
		for id, ch := range devices {
			listeners[id] = ch
		}
	}
	return listeners
}

func (l *listenerMap) Remove(chatID int, deviceID int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	devices, ok := l.data[chatID]
	if !ok {
		return
	}
	if ch, ok := devices[deviceID]; ok {
		close(ch)
		delete(devices, deviceID)
	}
	if len(devices) == 0 {
		delete(l.data, chatID)
	}
}

func (l *listenerMap) CloseAll() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for chatID, devices := range l.data {
		for deviceID, ch := range devices {
			close(ch)
			delete(devices, deviceID)
		}
		delete(l.data, chatID)
	}
}
