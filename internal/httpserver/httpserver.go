package httpserver

import (
	"context"
	"html/template"
	"net/http"
	"time"

	"github.com/xtra1n/local-messenger/internal/config"
	"github.com/xtra1n/local-messenger/internal/domain"
	"github.com/xtra1n/local-messenger/pkg/logger"
)

type Server struct {
	cfg         *config.Config
	log         *logger.Logger
	messenger   domain.Messenger
	userStore   domain.UserStore
	sessions    *sessionStore
	srv         *http.Server
	rateLimiter *SimpleTokenBucket
}

func New(cfg *config.Config, log *logger.Logger, m domain.Messenger, us domain.UserStore) *Server {
	s := &Server{
		cfg:         cfg,
		log:         log,
		messenger:   m,
		userStore:   us,
		sessions:    newSessionStore(),
		rateLimiter: NewSimpleTokenBucket(100, time.Minute),
	}

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("web"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	corsCfg := DefaultCORSConfig()
	if cfg.Server.Port == "80" || cfg.Server.Port == "443" {
		corsCfg.AllowOrigins = []string{"https://yourdomain.com"}
	}
	corsMiddleware := CORSMiddleware(corsCfg)

	apiMux := http.NewServeMux()

	apiMux.Handle("/message", RateLimitMiddleware(s.rateLimiter)(http.HandlerFunc(s.messageHandler)))
	apiMux.Handle("/healthz", http.HandlerFunc(s.healthHandler))
	apiMux.Handle("/metrics", http.HandlerFunc(s.metricsHandler))

	apiMux.Handle("/ws", s.authMiddleware(http.HandlerFunc(s.wsHandler)))

	apiMux.Handle("/debug/stream", RateLimitMiddleware(s.rateLimiter)(http.HandlerFunc(s.streamHandler)))

	mux.Handle("/", corsMiddleware(apiMux))

	mux.Handle("/main", s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie("session_token")
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		sess, ok := s.sessions.Get(c.Value)
		if !ok {
			s.sessions.ClearCookie(w)
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		tmpl, err := template.ParseFiles("web/index.html")
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("template error"))
			return
		}

		data := struct{ Username string }{Username: sess.Username}
		_ = tmpl.Execute(w, data)
	})))

	mux.Handle("/login", corsMiddleware(http.HandlerFunc(s.loginPageHandler)))
	mux.Handle("/signup", corsMiddleware(http.HandlerFunc(s.signupPageHandler)))

	s.srv = &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	return s
}

func (s *Server) Start() error {
	s.log.Info("HTTP server starting on ", s.cfg.Server.Port)
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
