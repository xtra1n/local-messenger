package httpserver

import (
	"encoding/hex"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

type session struct {
	Username string
	Expires  time.Time
}

type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]session
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]session),
	}
}

func (s *sessionStore) newToken() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func (s *sessionStore) Set(token, username string, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[token] = session{
		Username: username,
		Expires:  time.Now().Add(ttl),
	}
}

func (s *sessionStore) Get(token string) (session, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().After(sess.Expires) {
		return session{}, false
	}

	return sess, true
}

func (s *sessionStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
}

func (s *sessionStore) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   "",
		Path:    "/",
		Expires: time.Now().Add(-time.Hour),
	})
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" || r.URL.Path == "/signup" || r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}

		c, err := r.Cookie("session_token")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		if _, ok := s.sessions.Get(c.Value); !ok {
			s.sessions.ClearCookie(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
