package messenger

import (
	"testing"
)

func TestListenersMap_Get(t *testing.T) {
	lm := newListenersMap()

	ch1 := lm.Get(1, 100)
	ch2 := lm.Get(1, 101)
	ch3 := lm.Get(2, 100)

	if ch1 == ch2 {
		t.Fatal("Get() should return different channels for different deviceIDs")
	}
	if ch1 == ch3 {
		t.Fatal("Get() should return different channels for different chatIDs")
	}

	// Проверка, что каналы рабочие
	select {
	case ch1 <- Message{Text: "test"}:
	default:
		t.Fatal("channel should not be full")
	}
}

func TestListenersMap_Remove(t *testing.T) {
	lm := newListenersMap()

	lm.Get(1, 100)
	lm.Remove(1, 100)

	listenerss := lm.GetChatListeners(1)
	if len(listenerss) != 0 {
		t.Errorf("GetChatListenerss() = %d, want 0", len(listenerss))
	}
}

func TestListenersMap_GetChatListenerss(t *testing.T) {
	lm := newListenersMap()

	lm.Get(1, 100)
	lm.Get(1, 101)
	lm.Get(1, 102)
	lm.Get(2, 100)

	listenerss := lm.GetChatListeners(1)
	if len(listenerss) != 3 {
		t.Errorf("GetChatListenerss() = %d, want 3", len(listenerss))
	}

	listenerss2 := lm.GetChatListeners(2)
	if len(listenerss2) != 1 {
		t.Errorf("GetChatListenerss(2) = %d, want 1", len(listenerss2))
	}
}
