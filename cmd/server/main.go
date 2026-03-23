package main

import (
	"context"
	"database/sql"
	"os/signal"
	"syscall"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xtra1n/local-messenger/internal/config"
	"github.com/xtra1n/local-messenger/internal/httpserver"
	"github.com/xtra1n/local-messenger/internal/messenger"
	"github.com/xtra1n/local-messenger/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format, nil)

	log.Info("starting local-messenger",
		"host", cfg.Server.Host,
		"port", cfg.Server.Port,
		"db", cfg.Database.Path,
	)

	db, err := sql.Open("sqlite3", cfg.Database.Path)
	if err != nil {
		log.Fatal("failed to open sqlite db: ", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := initDB(db); err != nil {
		log.Fatal("failed to init sqlite schema", err)
	}

	store := messenger.NewSQLiteStore(db)
	userStore := messenger.NewSQLiteUserStore(db)

	m := messenger.New(log, store)
	srv := httpserver.New(cfg, log, m, userStore)

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
