package httpserver

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/xtra1n/local-messenger/internal/messenger"
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

func (s *Server) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFiles("web/login.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("template error"))
			return
		}
		tmpl.Execute(w, nil)
	case http.MethodPost:
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("username and password required"))
			return
		}

		ctx := r.Context()
		user, err := s.userStore.GetUserByUsername(ctx, username)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid credentials"))
			return
		}

		if !messenger.CheckPassword(password, user.PasswordHash) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid credentials"))
			return
		}

		token, err := s.sessions.newToken()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		s.sessions.Set(token, username, 24*time.Hour)

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) signupPageHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tmpl, err := template.ParseFiles("web/signup.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("template error"))
			return
		}
		tmpl.Execute(w, nil)
	case http.MethodPost:
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("username and password required"))
			return
		}

		ctx := r.Context()
		if err := s.userStore.CreateUser(ctx, username, password); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("could not create user"))
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)

	}
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
	c, err := r.Cookie("session_token")
	if err != nil || c.Value == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sess, ok := s.sessions.Get(c.Value)
	if !ok {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	q := r.URL.Query()
	q.Set("user", sess.Username)
	r.URL.RawQuery = q.Encode()

	s.messenger.HandleWS(w, r)
}
