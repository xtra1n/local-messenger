package logger

import (
	"io"
	"log/slog"
	"os"
)

type Logger struct {
	*slog.Logger
}

func New(level string, format string, w io.Writer) *Logger {
	if w == nil {
		w = os.Stdout
	}

	var handler slog.Handler
	lvl := new(slog.LevelVar)

	switch level {
	case "debug":
		lvl.Set(slog.LevelDebug)
	case "info":
		lvl.Set(slog.LevelInfo)
	case "warn":
		lvl.Set(slog.LevelWarn)
	case "error":
		lvl.Set(slog.LevelError)
	default:
		lvl.Set(slog.LevelInfo)
	}

	opts := &slog.HandlerOptions{
		Level: lvl,
	}

	if format == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

func (l *Logger) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

func (l *Logger) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

func (l *Logger) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

func (l *Logger) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

func (l *Logger) Fatal(msg string, args ...any) {
	l.Logger.Error(msg, args...)
	os.Exit(1)
}
