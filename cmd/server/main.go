package main

import (
	"context"
	"database/sql"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/xtra1n/local-messenger/internal/config"
	"github.com/xtra1n/local-messenger/internal/httpserver"
	"github.com/xtra1n/local-messenger/internal/infrastructure/store"
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

	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping database: ", err)
	}

	log.Info("database connected",
		"max_open", cfg.Database.MaxOpenConns,
		"max_idle", cfg.Database.MaxIdleConns,
	)

	if err := initDB(db); err != nil {
		log.Fatal("failed to init sqlite schema", err)
	}

	messageStore := store.NewSQLiteStore(db)
	userStore := store.NewSQLiteUserStore(db)

	m := messenger.New(log, messageStore)
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

func initDB(db *sql.DB) error {
	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS messages (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            chat INTEGER NOT NULL,
            text TEXT NOT NULL,
            by TEXT NOT NULL,
            at DATETIME NOT NULL
        );
    `); err != nil {
		return err
	}

	if _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at DATETIME NOT NULL
        );
    `); err != nil {
		return err
	}

	return nil
}
