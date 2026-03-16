package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/xtra1n/local-messenger/internal/config"
	"github.com/xtra1n/local-messenger/internal/messenger"
	"github.com/xtra1n/local-messenger/pkg/logger"
)

type Server struct {
	cfg       *config.Config
	log       *logger.Logger
	messenger messenger.Messenger
	userStore messenger.UserStore
	sessions  *sessionStore
	srv       *http.Server
}

func New(cfg *config.Config, log *logger.Logger, m messenger.Messenger, us messenger.UserStore) *Server {
	s := &Server{
		cfg:       cfg,
		log:       log,
		messenger: m,
		userStore: us,
		sessions:   newSessionStore(),
	}

	mux := http.NewServeMux()
	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/", s.authMiddleware(fileServer))

	mux.HandleFunc("/login", s.loginPageHandler)
	mux.HandleFunc("/signup", s.signupPageHandler)

	mux.HandleFunc("/healthz", s.healthHandler)
	mux.HandleFunc("/message", s.messageHandler)
	mux.HandleFunc("/metrics", s.metricsHandler)
	mux.HandleFunc("/debug/stream", s.streamHandler)
	mux.HandleFunc("/ws", s.wsHandler)

	s.srv = &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	s.log.Info("HTTP server starting on ", s.cfg.HTTPPort)
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("HTTP server shutting down...")
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return s.srv.Shutdown(ctx)
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
			s.sessions.ClearCoockie(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
