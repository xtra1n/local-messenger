package httpserver

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xtra1n/local-messenger/internal/messenger"
)

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) messageHandler(w http.ResponseWriter, r *http.Request) {
	s.messenger.AddMessage(w, r)
}

func (s *Server) metricsHandler(w http.ResponseWriter, r *http.Request) {
	s.messenger.MetricsHandler(w, r)
}

func (s *Server) loginPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/login.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("template error"))
		return
	}

	type viewData struct {
		Error    string
		Username string
	}

	switch r.Method {
	case http.MethodGet:
		_ = tmpl.Execute(w, nil)
		return

	case http.MethodPost:
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		data := viewData{Username: username}

		if username == "" || password == "" {
			w.WriteHeader(http.StatusBadRequest)
			data.Error = "Username и пароль обязательны"
			_ = tmpl.Execute(w, data)
			return
		}

		ctx := r.Context()
		user, err := s.userStore.GetUserByUsername(ctx, username)
		if err != nil {
			s.log.Error("login: GetUserByUsername error: ", err)
			w.WriteHeader(http.StatusUnauthorized)
			data.Error = "Неверное имя пользователя или пароль"
			_ = tmpl.Execute(w, data)
			return
		}

		if !messenger.CheckPassword(password, user.PasswordHash) {
			w.WriteHeader(http.StatusUnauthorized)
			data.Error = "Неверное имя пользователя или пароль"
			_ = tmpl.Execute(w, data)
			return
		}

		token, err := s.sessions.newToken()
		if err != nil {
			s.log.Error("login: newToken error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			data.Error = "Внутренняя ошибка, попробуйте позже"
			_ = tmpl.Execute(w, data)
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
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) signupPageHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("web/signup.html")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("template error"))
		return
	}

	type viewData struct {
		Error    string
		Username string
	}

	switch r.Method {
	case http.MethodGet:
		_ = tmpl.Execute(w, nil)
		return

	case http.MethodPost:
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		password2 := r.FormValue("password2")

		data := viewData{Username: username}

		if username == "" || password == "" || password2 == "" {
			w.WriteHeader(http.StatusBadRequest)
			data.Error = "Все поля обязательны"
			_ = tmpl.Execute(w, data)
			return
		}

		if password != password2 {
			w.WriteHeader(http.StatusBadRequest)
			data.Error = "Пароли не совпадают"
			_ = tmpl.Execute(w, data)
			return
		}

		ctx := r.Context()
		if err := s.userStore.CreateUser(ctx, username, password); err != nil {
			s.log.Error("signup: CreateUser error: ", err)
			w.WriteHeader(http.StatusBadRequest)
			data.Error = "Не удалось создать пользователя (возможно, имя занято)"
			_ = tmpl.Execute(w, data)
			return
		}

		http.Redirect(w, r, "/login", http.StatusFound)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *Server) streamHandler(w http.ResponseWriter, r *http.Request) {
	chatStr := r.URL.Query().Get("chat")
	if chatStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("chat query param required"))
		return
	}

	chatID, err := strconv.Atoi(chatStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid chat id"))
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
