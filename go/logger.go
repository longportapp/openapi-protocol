package protocol

import (
	"log"
	"strings"
)

type LogLevel int

const (
	LevelDebug LogLevel = -1
	LevelInfo  LogLevel = iota
	LevelWarn
	LevelError
)

// Logger provider log methods
type Logger interface {
	SetLevel(string)
	Info(msg string)
	Error(msg string)
	Warn(msg string)
	Debug(msg string)
	Infof(msg string, args ...any)
	Errorf(msg string, args ...any)
	Warnf(msg string, args ...any)
	Debugf(msg string, args ...any)
}

type DefaultLogger struct {
	lvl LogLevel
}

func (l *DefaultLogger) SetLevel(lvl string) {
	lvl = strings.ToLower(lvl)

	switch lvl {
	case "info":
		l.lvl = LevelInfo
	case "debug":
		l.lvl = LevelDebug
	case "warn":
		l.lvl = LevelWarn
	case "error":
		l.lvl = LevelError
	}
}

func (l *DefaultLogger) Info(msg string) {
	if l.lvl <= LevelInfo {
		log.Println("[INFO]", msg)
	}

}

func (l *DefaultLogger) Infof(msg string, args ...any) {
	if l.lvl <= LevelInfo {
		log.Printf("[INFO] "+msg, args...)
	}

}

func (l *DefaultLogger) Error(msg string) {
	if l.lvl <= LevelError {
		log.Println("[ERR]", msg)
	}
}

func (l *DefaultLogger) Errorf(msg string, args ...any) {
	if l.lvl <= LevelError {
		log.Printf("[ERR] "+msg, args...)
	}
}

func (l *DefaultLogger) Debug(msg string) {
	if l.lvl <= LevelDebug {
		log.Println("[DEBUG]", msg)
	}
}

func (l *DefaultLogger) Debugf(msg string, args ...any) {
	if l.lvl <= LevelDebug {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

func (l *DefaultLogger) Warn(msg string) {
	if l.lvl <= LevelWarn {
		log.Println("[WARN]", msg)
	}
}

func (l *DefaultLogger) Warnf(msg string, args ...any) {
	if l.lvl <= LevelWarn {
		log.Printf("[WARN] "+msg, args...)
	}
}
