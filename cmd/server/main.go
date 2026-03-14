package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/xtra1n/local-messenger/internal/config"
	"github.com/xtra1n/local-messenger/internal/httpserver"
	"github.com/xtra1n/local-messenger/internal/messenger"
	"github.com/xtra1n/local-messenger/pkg/logger"
)

func main() {
	cfg := config.Load()

	log := logger.New(cfg.LogLevel)

	m := messenger.New(log)
	srv := httpserver.New(cfg, log, m)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := m.Run(ctx); err != nil {
			log.Error("messenger stopped: ", err)
		}
	}()

	go func() {
		if err := srv.Start(); err != nil {
			log.Error("http server error: ", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	if err := srv.Shutdown(context.Background()); err != nil {
		log.Error("shutdown error: ", err)
	}

	log.Info("server stopped")
}
