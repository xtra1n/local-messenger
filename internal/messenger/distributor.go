package messenger

import "context"

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
