package messenger

import (
	"context"

	"github.com/xtra1n/local-messenger/internal/domain"
)

func (m *messenger) distributor(ctx context.Context) {
	m.log.Info("distributor worker started")

	for {
		select {
		case <-ctx.Done():
			m.log.Info("distributor worker stopping")
			return
		case msg := <-m.input:
			m.handleIncomingMessege(msg)
		}
	}
}

func (m *messenger) handleIncomingMessege(msg domain.Message) {
	listeners := m.listeners.GetChatListeners(msg.Chat)
	if len(listeners) == 0 {
		m.log.Debug("no listeners for chat ", msg.Chat)
		return
	}

	m.dispatchToListners(msg, listeners)
}

func (m *messenger) dispatchToListners(msg domain.Message, listeners map[int]chan domain.Message) {
	for deviceID, ch := range listeners {
		select {
		case ch <- msg:
			m.metrics.MessagesSampled.Add(1)
		default:
			m.log.Debug("listener channel full, chat=", msg.Chat, " device=", deviceID)
		}
	}
}
