package logger

import (
	"io"
	"log"
	"os"
)

type Logger struct {
	*log.Logger
	level string
}

func New(level string) *Logger {
	return &Logger{
		Logger: log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds),
		level:  level,
	}
}

func (l *Logger) Debug(v ...any) {
	if l.level == "debug" {
		l.Logger.SetPrefix("[DEBUG] ")
		l.Println(v...)
	}
}

func (l *Logger) Info(v ...any) {
	l.Logger.SetPrefix("[INFO] ")
	l.Println(v...)
}

func (l *Logger) Error(v ...any) {
	l.Logger.SetPrefix("[ERROR] ")
	l.Println(v...)
}

func (l *Logger) SetOutput(w io.Writer) {
	l.Logger.SetOutput(w)
}
