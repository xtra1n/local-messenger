package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) messageHandler(w http.ResponseWriter, r *http.Request) {
	s.messenger.AddMessage(w, r)
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.messenger.MetricsHandler(w, r)
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	chatStr := r.URL.Query().Get("chat")
	if chatStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("chat query param required"))
		return
	}

	chatID, err := strconv.Atoi(chatStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid chat id"))
		return
	}

	ch := s.messenger.Subscribe(chatID, 1)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	enc := json.NewEncoder(w)

	for {
		select {
		case msg := <-ch:
			if err := enc.Encode(msg); err != nil {
				return
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Minute):
			return
		}
	}
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	s.messenger.HandleWS(w, r)
}
